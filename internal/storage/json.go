package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thedittmer/rss-reader/internal/models"
)

type Storage struct {
	dataDir string
}

func NewStorage() (*Storage, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Create .rss-reader directory in user's home
	dataDir := filepath.Join(homeDir, ".rss-reader")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	fmt.Printf("Using storage directory: %s\n", dataDir)
	return &Storage{dataDir: dataDir}, nil
}

func (s *Storage) SaveProfile(profile *models.UserProfile) error {
	log.Printf("Saving profile with %d interests", len(profile.Interests))
	path := filepath.Join(s.dataDir, "profile.json")
	tempPath := path + ".tmp"

	// Update LastUpdated time
	profile.LastUpdated = time.Now()

	// Marshal with pretty printing for readability
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling profile: %w", err)
	}

	// Write to temporary file first
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("error writing temporary profile: %w", err)
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("error saving profile: %w", err)
	}

	fmt.Printf("Profile saved successfully with %d interests\n", len(profile.Interests))
	return nil
}

func (s *Storage) LoadProfile() (*models.UserProfile, error) {
	path := filepath.Join(s.dataDir, "profile.json")

	profile := models.NewUserProfile()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty profile file
			if err := s.SaveProfile(profile); err != nil {
				return nil, fmt.Errorf("error creating initial profile: %w", err)
			}
			return profile, nil
		}
		return nil, fmt.Errorf("error reading profile: %w", err)
	}

	if err := json.Unmarshal(data, profile); err != nil {
		return nil, fmt.Errorf("error parsing profile: %w", err)
	}

	// Validate and clean up profile data
	if profile.Interests == nil {
		profile.Interests = make(map[string]float64)
	}
	if profile.ReadArticles == nil {
		profile.ReadArticles = make(map[string]bool)
	}
	if profile.LastUpdated.IsZero() {
		profile.LastUpdated = time.Now()
	}

	return profile, nil
}

func (s *Storage) SaveFeeds(feeds []string) error {
	path := filepath.Join(s.dataDir, "feeds.txt")

	// Create the content with comments
	content := "# RSS Feed URLs (one per line)\n" +
		"# Lines starting with # are comments\n" +
		"# Example: https://example.com/feed.xml\n\n"

	for _, feed := range feeds {
		content += feed + "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("error saving feeds: %w", err)
	}

	fmt.Printf("Feeds saved successfully to: %s\n", path)
	return nil
}

func (s *Storage) LoadFeeds() ([]string, error) {
	path := filepath.Join(s.dataDir, "feeds.txt")

	// Check if file exists, if not create with default feeds
	if _, err := os.Stat(path); os.IsNotExist(err) {
		defaultFeeds := []string{
			"https://lessnews.dev/rss.xml",
			"https://blog.golang.org/feed.atom",
			"https://news.ycombinator.com/rss",
			"https://dev.to/feed",
		}
		if err := s.SaveFeeds(defaultFeeds); err != nil {
			return nil, fmt.Errorf("error creating default feeds file: %w", err)
		}
		return defaultFeeds, nil
	}

	// Read and parse the feeds file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading feeds file: %w", err)
	}

	var feeds []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			feeds = append(feeds, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing feeds file: %w", err)
	}

	fmt.Printf("Loaded %d feeds from: %s\n", len(feeds), path)
	return feeds, nil
}
