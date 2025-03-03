package config

import (
	"bufio"
	"os"
	"strings"
)

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
