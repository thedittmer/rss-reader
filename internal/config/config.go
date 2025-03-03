package config

import (
	"bufio"
	"os"
	"strings"
	"time"
)

type Config struct {
	Theme struct {
		Dark        bool
		AccentColor string
	}
	Behavior struct {
		AutoRefreshInterval time.Duration
		MaxArticlesPerFeed  int
		DefaultPageSize     int
	}
	Display struct {
		CompactView    bool
		ShowReadStatus bool
		DateFormat     string
	}
	Keyboard struct {
		NextPage    string `json:"nextPage"`
		PrevPage    string `json:"prevPage"`
		OpenArticle string `json:"openArticle"`
		Back        string `json:"back"`
	}
}

func LoadFeedsFromFile(filename string) ([]string, error) {
	var feeds []string

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" && !strings.HasPrefix(url, "#") {
			feeds = append(feeds, url)
		}
	}

	return feeds, scanner.Err()
}

func LoadConfig() (*Config, error) {
	// Implementation
}

func SaveConfig(cfg *Config) error {
	// Implementation
}
