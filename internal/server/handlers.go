package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"dev-journal/internal/config"
	"dev-journal/internal/content"
	"dev-journal/internal/database"
)

type Server struct {
	db    *database.DB
	cfg   *config.Config
	md    goldmark.Markdown
	tmpls *template.Template
}

func New(db *database.DB, cfg *config.Config) *Server {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	tmpls := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Server{
		db:    db,
		cfg:   cfg,
		md:    md,
		tmpls: tmpls,
	}
}

func (s *Server) RegisterRoutes(r *chi.Mux) {
	// Middleware for admin routes
	r.Use(s.AdminAuthMiddleware)

	// Public routes
	r.Get("/", s.handleHomepage)
	r.Get("/*", s.handlePageOrAsset)
	r.Post("/webhook", s.handleWebhook)

	// Admin routes
	r.Get(s.cfg.AdminLoginPath, s.handleAdminLogin)
	r.Post(s.cfg.AdminLoginPath, s.handleAdminLoginAttempt)
	r.Route("/admin", func(r chi.Router) {
		r.Use(s.RequireAdmin) // Protect all /admin routes
		r.Get("/dashboard", s.handleAdminDashboard)
		r.Post("/pages/{pagePath}/toggle", s.handleAdminToggleVisibility)
		r.Get("/logout", s.handleAdminLogout)
	})
}

// render executes the given template with the provided data.
func (s *Server) render(w http.ResponseWriter, name string, data map[string]interface{}) {
	navPages, err := s.db.GetVisiblePages()
	if err != nil {
		http.Error(w, "Could not fetch navigation", http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = make(map[string]interface{})
	}

	data["NavPages"] = navPages
	data["Theme"] = s.cfg.Theme

	err = s.tmpls.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error rendering template %s: %v", name, err)
	}
}

func (s *Server) handleHomepage(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "home.md")
}

func (s *Server) handlePageOrAsset(w http.ResponseWriter, r *http.Request) {
	pagePath := chi.URLParam(r, "*")

	// If it's an image or other static asset from the content repo
	if strings.HasPrefix(pagePath, "img/") || strings.HasPrefix(pagePath, "assets/") {
		// This assumes your markdown references images like `/img/my-image.png`
		http.ServeFile(w, r, filepath.Join(s.cfg.ContentPath, pagePath))
		return
	}

	// Otherwise, assume it's a markdown page
	// Ensure path ends with .md for lookup
	if !strings.HasSuffix(pagePath, ".md") {
		if pagePath == "" { // Handle root case
			s.renderPage(w, r, "home.md")
			return
		}
		pagePath = pagePath + ".md"
	}
	s.renderPage(w, r, pagePath)
}

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, pagePath string) {
	page, err := s.db.GetPageByPath(pagePath)
	if err != nil || !page.IsVisible {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	// Read the markdown file
	file, err := http.Dir(s.cfg.ContentPath).Open(page.Path)
	if err != nil {
		http.Error(w, "Could not read page content", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	mdContent, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read page content", http.StatusInternalServerError)
		return
	}

	// Convert markdown to HTML
	var buf bytes.Buffer
	if err := s.md.Convert(mdContent, &buf); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
		return
	}

	// Increment visit count (best effort)
	go s.db.IncrementVisitCount(page.Path)

	data := map[string]interface{}{
		"Title":   page.Title,
		"Content": template.HTML(buf.String()),
	}
	s.render(w, "page.html", data)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// 1. Validate Signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Missing signature", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusInternalServerError)
		return
	}

	mac := hmac.New(sha256.New, []byte(s.cfg.GithubWebhookSecret))
	mac.Write(body)
	expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// 2. Check Event Type
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// We only care about pushes to the main/master branch.
	// You might want to make the branch name configurable.
	ref, ok := payload["ref"].(string)
	if !ok || (ref != "refs/heads/main" && ref != "refs/heads/master") {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Payload received, but not for main/master branch. Ignoring."))
		return
	}

	// 3. Trigger Git Pull and Sync
	log.Println("Webhook validated. Triggering content update...")
	go func() {
		if err := content.PullRepo(s.cfg); err != nil {
			log.Printf("ERROR: Failed to pull repo: %v", err)
			return
		}
		log.Println("Content repository pulled successfully.")

		if err := content.Sync(s.cfg.ContentPath, s.db); err != nil {
			log.Printf("ERROR: Failed to sync content after pull: %v", err)
			return
		}
		log.Println("Content sync complete.")
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Webhook accepted. Processing update."))
}

// --- Admin Handlers ---
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	s.render(w, "admin_login.html", map[string]interface{}{
		"Title": "Admin Login",
	})
}

func (s *Server) handleAdminLoginAttempt(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	password := r.FormValue("password")

	if password == s.cfg.AdminSecret {
		expiration := time.Now().Add(24 * time.Hour)
		cookie := http.Cookie{
			Name:     "admin_session",
			Value:    "logged_in",
			Expires:  expiration,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	} else {
		data := map[string]interface{}{
			"Title": "Admin Login",
			"Error": "Invalid password",
		}
		s.render(w, "admin_login.html", data)
	}
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, s.cfg.AdminLoginPath, http.StatusFound)
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	pages, err := s.db.GetAllPages()
	if err != nil {
		http.Error(w, "Could not fetch pages", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Title": "Admin Dashboard",
		"Pages": pages,
	}
	s.render(w, "admin_dashboard.html", data)
}

func (s *Server) handleAdminToggleVisibility(w http.ResponseWriter, r *http.Request) {
	// Note: Chi URLParam decodes the path, which is what we need.
	pagePath := chi.URLParam(r, "*")
	if pagePath == "" {
		http.Error(w, "Page path is required", http.StatusBadRequest)
		return
	}

	// The path from the URL might not have .md, ensure it does for DB lookup
	if !strings.HasSuffix(pagePath, ".md") {
		pagePath += ".md"
	}

	err := s.db.ToggleVisibility(pagePath)
	if err != nil {
		http.Error(w, "Failed to toggle visibility", http.StatusInternalServerError)
		return
	}

	// HTMX response: redirect back to the dashboard to see the change
	w.Header().Set("HX-Redirect", "/admin/dashboard")
	w.WriteHeader(http.StatusOK)
}
