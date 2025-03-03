package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mmcdole/gofeed"
)

type FeedItem struct {
	Title       string
	Description string
	Link        string
	Published   string
	FeedSource  string
}

func main() {
	// Replace the hardcoded feeds list with loading from file
	feeds, err := loadFeedsFromFile("feeds.txt")
	if err != nil {
		fmt.Printf("Error loading feeds: %v\n", err)
		// Fallback to default feed
		feeds = []string{"https://lessnews.dev/rss.xml"}
	}

	// Parse feeds concurrently
	var allItems []FeedItem
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, feedURL := range feeds {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			items := parseFeed(url)

			mu.Lock()
			allItems = append(allItems, items...)
			mu.Unlock()
		}(feedURL)
	}
	wg.Wait()

	// Search functionality
	searchFeeds(allItems)
}

func parseFeed(feedURL string) []FeedItem {
	var items []FeedItem

	// Validate feed URL format
	if !strings.HasPrefix(feedURL, "http://") && !strings.HasPrefix(feedURL, "https://") {
		fmt.Printf("Error: Invalid feed URL format %s (must start with http:// or https://)\n", feedURL)
		return items
	}

	// Check if URL ends with common feed extensions
	isValidFeed := strings.HasSuffix(feedURL, ".xml") ||
		strings.HasSuffix(feedURL, ".rss") ||
		strings.HasSuffix(feedURL, "/feed") ||
		strings.HasSuffix(feedURL, "/rss") ||
		strings.HasSuffix(feedURL, "/feed.xml") ||
		strings.HasSuffix(feedURL, "/rss.xml")

	if !isValidFeed && !strings.Contains(feedURL, "feed") && !strings.Contains(feedURL, "rss") {
		fmt.Printf("Warning: URL %s might not be a valid RSS feed\n", feedURL)
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		fmt.Printf("Error parsing feed %s: %v\n", feedURL, err)
		return items
	}

	if len(feed.Items) == 0 {
		fmt.Printf("Warning: Feed %s contains no items\n", feedURL)
		return items
	}

	fmt.Printf("Successfully parsed feed: %s (%d items)\n", feed.Title, len(feed.Items))

	for _, item := range feed.Items {
		items = append(items, FeedItem{
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			Published:   item.Published,
			FeedSource:  feed.Title,
		})
	}

	return items
}

func searchFeeds(items []FeedItem) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nEnter search term (or 'exit' to quit): ")
		searchTerm, _ := reader.ReadString('\n')
		searchTerm = strings.TrimSpace(searchTerm)

		if strings.ToLower(searchTerm) == "exit" {
			break
		}

		fmt.Printf("\nSearching for: %s\n\n", searchTerm)
		found := false

		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Title), strings.ToLower(searchTerm)) ||
				strings.Contains(strings.ToLower(item.Description), strings.ToLower(searchTerm)) {
				found = true
				fmt.Printf("Source: %s\n", item.FeedSource)
				fmt.Printf("Title: %s\n", item.Title)
				fmt.Printf("Published: %s\n", item.Published)
				fmt.Printf("Link: %s\n", item.Link)
				fmt.Printf("Description: %s\n\n", item.Description)
			}
		}

		if !found {
			fmt.Println("No results found.")
		}
	}
}

func loadFeedsFromFile(filename string) ([]string, error) {
	var feeds []string
	var invalidFeeds []string

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		url := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if url == "" || strings.HasPrefix(url, "#") {
			continue
		}

		// Basic URL validation
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			invalidFeeds = append(invalidFeeds, fmt.Sprintf("Line %d: %s (invalid URL format)", lineNum, url))
			continue
		}

		feeds = append(feeds, url)
	}

	// Report any invalid feeds found
	if len(invalidFeeds) > 0 {
		fmt.Println("\nWarning: Found invalid feeds in configuration:")
		for _, invalid := range invalidFeeds {
			fmt.Printf("- %s\n", invalid)
		}
		fmt.Println()
	}

	return feeds, scanner.Err()
}
