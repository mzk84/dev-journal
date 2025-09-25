package content

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dev-journal/internal/config"
	"dev-journal/internal/database"
)

// CloneRepo clones the git repository if the content directory doesn't exist.
func CloneRepo(cfg *config.Config) error {
	if _, err := os.Stat(cfg.ContentPath); !os.IsNotExist(err) {
		log.Println("Content directory already exists. Skipping initial clone.")
		// Optionally, you could do a pull here to ensure it's up to date on start
		return PullRepo(cfg)
	}

	log.Printf("Cloning repository %s into %s...", cfg.GitRepoURL, cfg.ContentPath)
	// Configure SSH command to use the specific deploy key
	sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=no", cfg.GitSSHKeyPath)
	cmd := exec.Command("git", "clone", cfg.GitRepoURL, cfg.ContentPath)
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s\n%v", string(output), err)
	}

	return nil
}

// PullRepo pulls the latest changes from the git repository.
func PullRepo(cfg *config.Config) error {
	log.Println("Pulling latest changes from repository...")
	sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=no", cfg.GitSSHKeyPath)
	cmd := exec.Command("git", "pull")
	cmd.Dir = cfg.ContentPath
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s\n%v", string(output), err)
	}
	return nil
}

// Sync walks the content directory and ensures all .md files are in the database.
func Sync(contentPath string, db *database.DB) error {
	log.Println("Starting content sync with database...")
	err := filepath.WalkDir(contentPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// We only care about .md files
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			// Get the relative path from the content directory
			relPath, err := filepath.Rel(contentPath, path)
			if err != nil {
				return err
			}
			// Use forward slashes for URL and DB consistency
			relPath = filepath.ToSlash(relPath)

			log.Printf("Found markdown file: %s", relPath)
			if err := db.UpsertPage(relPath); err != nil {
				log.Printf("Failed to upsert page %s: %v", relPath, err)
				// We continue even if one fails
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking content directory: %w", err)
	}
	log.Println("Content sync finished.")
	return nil
}
