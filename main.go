package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type FeedItem struct {
	Title       string
	Description string
	Link        string
	Published   string
	FeedSource  string
}

type SearchOptions struct {
	StartDate time.Time
	EndDate   time.Time
	Source    string
}

type UserProfile struct {
	Interests    map[string]float64 // word -> weight
	ReadArticles map[string]bool    // article URLs that have been read
	LastUpdated  time.Time
}

type ArticleScore struct {
	Item  FeedItem
	Score float64
}

const (
	userProfileFile = "user_profile.json"
	maxInterests    = 100  // Maximum number of interests to track
	minWeight       = 0.1  // Minimum weight to keep an interest
	decayFactor     = 0.95 // How much to decay weights over time
)

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
	profile := loadUserProfile()
	defer profile.save()

	for {
		fmt.Print("\nOptions:\n")
		fmt.Print("1. Search articles\n")
		fmt.Print("2. Show recommended articles\n")
		fmt.Print("3. View my interests\n")
		fmt.Print("4. Exit\n")
		fmt.Print("\nEnter choice (1-4): ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			searchArticles(items, &profile)
		case "2":
			showRecommendations(items, &profile)
		case "3":
			showInterests(&profile)
		case "4":
			return
		default:
			fmt.Println("Invalid choice")
		}
	}
}

func searchArticles(items []FeedItem, profile *UserProfile) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\nEnter search term: ")
	searchTerm, _ := reader.ReadString('\n')
	searchTerm = strings.TrimSpace(searchTerm)

	results := advancedSearch(items, searchTerm, SearchOptions{})
	displayResults(results, profile)
}

func showRecommendations(items []FeedItem, profile *UserProfile) {
	// Score all items based on user interests
	var scored []ArticleScore

	for _, item := range items {
		if profile.ReadArticles[item.Link] {
			continue // Skip already read articles
		}

		score := 0.0
		text := item.Title + " " + item.Description
		words := extractKeywords(text)

		for _, word := range words {
			if weight, exists := profile.Interests[word]; exists {
				score += weight
			}
		}

		if score > 0 {
			scored = append(scored, ArticleScore{item, score})
		}
	}

	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Display top recommendations
	fmt.Println("\nRecommended articles:")
	for i, article := range scored {
		if i >= 10 { // Show top 10
			break
		}
		displayArticle(article.Item, profile)
	}
}

func showInterests(profile *UserProfile) {
	fmt.Println("\nYour interests (word: weight):")

	// Sort interests by weight
	type weightedWord struct {
		word   string
		weight float64
	}

	sorted := make([]weightedWord, 0, len(profile.Interests))
	for word, weight := range profile.Interests {
		sorted = append(sorted, weightedWord{word, weight})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].weight > sorted[j].weight
	})

	for _, ww := range sorted {
		fmt.Printf("%s: %.2f\n", ww.word, ww.weight)
	}
}

func displayResults(items []FeedItem, profile *UserProfile) {
	for _, item := range items {
		displayArticle(item, profile)
	}
}

func displayArticle(item FeedItem, profile *UserProfile) {
	fmt.Printf("\nSource: %s\n", item.FeedSource)
	fmt.Printf("Title: %s\n", item.Title)
	fmt.Printf("Published: %s\n", item.Published)
	fmt.Printf("Link: %s\n", item.Link)
	fmt.Printf("Description: %s\n", item.Description)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nMark as interesting? (y/n/q): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "q" {
		return
	}

	if response == "y" {
		// Update user interests based on this article
		profile.updateInterests(item.Title + " " + item.Description)
		profile.ReadArticles[item.Link] = true
		fmt.Println("Added to your interests!")
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

func loadUserProfile() UserProfile {
	profile := UserProfile{
		Interests:    make(map[string]float64),
		ReadArticles: make(map[string]bool),
		LastUpdated:  time.Now(),
	}

	data, err := os.ReadFile(userProfileFile)
	if err == nil {
		json.Unmarshal(data, &profile)
	}

	return profile
}

func (p UserProfile) save() error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(userProfileFile, data, 0644)
}

func (p *UserProfile) updateInterests(text string) {
	// Extract important words (simple implementation)
	words := extractKeywords(text)

	// Update weights
	for _, word := range words {
		p.Interests[word] = p.Interests[word] + 1.0
	}

	// Decay old interests
	timeSinceUpdate := time.Since(p.LastUpdated)
	decayPeriods := timeSinceUpdate.Hours() / 24 // daily decay
	decayMultiplier := math.Pow(decayFactor, decayPeriods)

	for word, weight := range p.Interests {
		p.Interests[word] = weight * decayMultiplier
		if p.Interests[word] < minWeight {
			delete(p.Interests, word)
		}
	}

	// Trim to max interests
	if len(p.Interests) > maxInterests {
		// Remove lowest weighted interests
		weights := make([]float64, 0, len(p.Interests))
		for _, w := range p.Interests {
			weights = append(weights, w)
		}
		sort.Float64s(weights)
		threshold := weights[len(weights)-maxInterests]

		for word, weight := range p.Interests {
			if weight < threshold {
				delete(p.Interests, word)
			}
		}
	}

	p.LastUpdated = time.Now()
}

func extractKeywords(text string) []string {
	// Convert to lowercase and split into words
	words := strings.Fields(strings.ToLower(text))

	// Filter out common words and short terms
	keywords := make([]string, 0)
	for _, word := range words {
		if len(word) > 3 && !isCommonWord(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "that": true, "with": true,
		"this": true, "from": true, "your": true, "have": true, "are": true,
		// Add more common words as needed
	}
	return commonWords[word]
}

func advancedSearch(items []FeedItem, term string, options SearchOptions) []FeedItem {
	// Implementation of advanced search logic based on the term and options
	// This is a placeholder and should be replaced with the actual implementation
	return items
}
