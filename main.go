package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmcdole/gofeed"
	"github.com/thedittmer/rss-reader/internal/models"
	"github.com/thedittmer/rss-reader/internal/storage"
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

// Add this new type for search results
type SearchResult struct {
	Item       FeedItem
	Matches    []string // Snippets showing search term in context
	MatchCount int
}

const (
	userProfileFile = "user_profile.json"
	maxInterests    = 100  // Maximum number of interests to track
	minWeight       = 0.1  // Minimum weight to keep an interest
	decayFactor     = 0.95 // How much to decay weights over time
)

// Add these style definitions at the top level
var (
	// Base styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD"))

	searchPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#874BFD")).
				Bold(true)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB"))

	textStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	linkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0087BD")).
			Underline(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98FB98"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			SetString("───────────────")

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#874BFD"))
)

// Define weightedWord type at package level
type weightedWord struct {
	word   string
	weight float64
}

func main() {
	// Initialize signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Handle interrupt in a separate goroutine
	go func() {
		<-c
		fmt.Println("\nReceived interrupt signal. Saving and exiting...")
		os.Exit(0)
	}()

	// Initialize storage
	store, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Load user profile
	profile, err := store.LoadProfile()
	if err != nil {
		log.Fatalf("Failed to load profile: %v", err)
	}

	// Keep just this one call
	debugProfile(profile)

	// Load feeds
	feeds, err := loadFeedsFromFile("feeds.txt")
	if err != nil {
		fmt.Printf("Error loading feeds: %v\n", err)
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

	// Start the main program loop
	searchFeeds(allItems, profile, store)
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

func searchFeeds(items []FeedItem, profile *models.UserProfile, store *storage.Storage) {
	reader := bufio.NewReader(os.Stdin)

	for {
		clearScreen()

		searchUI := []string{
			"📚 RSS Reader",
			"───────────────",
			"",
			"1. 🔍 Search Articles",
			"2. ⭐ View Recommended",
			"3. 📋 View Interests",
			"4. 📑 Manage Feeds",
			"5. 🚪 Exit",
			"",
			"Enter your choice (1-5)",
		}

		fmt.Println(appStyle.Render(strings.Join(searchUI, "\n")))
		fmt.Print(searchPromptStyle.Render("→ "))

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			searchArticles(items, profile, store)
		case "2":
			showRecommendations(items, profile, store)
		case "3":
			showInterests(profile, store)
		case "4":
			manageFeeds(store)
		case "5":
			clearScreen()
			fmt.Println(appStyle.Render("Thanks for using RSS Reader! 👋"))
			return
		}
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func showSpinner(message string, duration time.Duration) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	startTime := time.Now()

	for time.Since(startTime) < duration {
		for _, frame := range spinner {
			fmt.Printf("\r%s %s", spinnerStyle.Render(frame), message)
			time.Sleep(50 * time.Millisecond)
		}
	}
	fmt.Println()
}

func searchArticles(items []FeedItem, profile *models.UserProfile, store *storage.Storage) {
	for {
		clearScreen()

		// Show search interface
		searchUI := []string{
			"🔍 RSS Search",
			"───────────────",
			"",
			"Enter search term or commands:",
			"  • Type your search terms",
			"  • Use 'exit' to return to main menu",
			"",
		}

		fmt.Println(appStyle.Render(strings.Join(searchUI, "\n")))

		// Get search input
		fmt.Print(searchPromptStyle.Render("→ "))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		searchTerm := strings.TrimSpace(input)

		if searchTerm == "exit" {
			return
		}

		if searchTerm == "" {
			continue
		}

		// Show searching animation
		showSpinner("Searching articles...", 800*time.Millisecond)

		// Perform search
		var results []FeedItem
		for _, item := range items {
			itemText := strings.ToLower(item.Title + " " + item.Description)
			searchText := strings.ToLower(searchTerm)

			if strings.Contains(itemText, searchText) {
				results = append(results, item)
			}
		}

		// Clear screen for results
		clearScreen()

		// Show results header
		fmt.Println(appStyle.Render(fmt.Sprintf(
			"Search Results for: %s\nFound %d articles\n",
			searchPromptStyle.Render(searchTerm),
			len(results))))

		if len(results) == 0 {
			fmt.Println(infoStyle.Render("No matching articles found."))
			fmt.Println("\nPress Enter to search again...")
			reader.ReadString('\n')
			continue
		}

		// Display results
		displayResults(results, profile, store)

		// Show options
		fmt.Println(appStyle.Render("\nOptions:"))
		fmt.Println("• Enter article number to mark as interesting")
		fmt.Println("• Press Enter to search again")
		fmt.Println("• Type 'exit' to return to main menu")
		fmt.Print(searchPromptStyle.Render("\n→ "))

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "exit" {
			return
		}

		// Handle article selection
		if num, err := strconv.Atoi(choice); err == nil && num > 0 && num <= len(results) {
			selected := results[num-1]
			profile.UpdateInterests(selected.Title + " " + selected.Description)
			profile.ReadArticles[selected.Link] = true

			// Show confirmation
			fmt.Println(infoStyle.Render("\n✨ Added to your interests!"))
			showSpinner("Updating recommendations...", 500*time.Millisecond)
		}
	}
}

func showRecommendations(items []FeedItem, profile *models.UserProfile, store *storage.Storage) {
	fmt.Println(titleStyle.Render("\n🎯 Recommended Articles"))
	fmt.Println(dividerStyle.Render())

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

	if len(scored) == 0 {
		fmt.Println(errorStyle.Render("No recommendations yet! Try marking some articles as interesting."))
		fmt.Print(promptStyle.Render("\nPress Enter to continue..."))
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	// Display top recommendations
	for i, article := range scored {
		if i >= 10 {
			break
		}
		fmt.Printf("\n%s #%d (Score: %.2f)\n",
			subtitleStyle.Render("Recommendation"),
			i+1,
			article.Score)
		displayArticle(article.Item, profile, store)
	}
}

func showInterests(profile *models.UserProfile, store *storage.Storage) {
	for {
		clearScreen()

		header := []string{
			"📋 Interest Management",
			"───────────────",
			"",
			"Your current interests:",
			"",
		}

		fmt.Println(appStyle.Render(strings.Join(header, "\n")))

		// Display current interests
		var sorted []weightedWord
		for word, weight := range profile.Interests {
			sorted = append(sorted, weightedWord{word, weight})
		}

		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].weight > sorted[j].weight
		})

		if len(sorted) == 0 {
			fmt.Println(infoStyle.Render("No interests yet. Try marking some articles as interesting!"))
		} else {
			for i, ww := range sorted {
				fmt.Printf("%d. %s %s\n",
					i+1,
					titleStyle.Render(ww.word),
					infoStyle.Render(fmt.Sprintf("(%.2f)", ww.weight)))
			}
		}

		options := []string{
			"",
			"Options:",
			"  [a] Add new interest",
			"  [r] Remove interest",
			"  [m] Modify weight",
			"  [c] Clear all interests",
			"  [s] Save changes",
			"  [x] Return to main menu",
			"",
			"Enter option [a/r/m/c/s/x]:",
		}

		fmt.Println(appStyle.Render(strings.Join(options, "\n")))
		fmt.Print(searchPromptStyle.Render("→ "))

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(strings.ToLower(choice))

		switch choice {
		case "a":
			addInterest(profile, store, reader)
		case "r":
			removeInterest(profile, store, reader, sorted)
		case "m":
			modifyInterest(profile, store, reader, sorted)
		case "c":
			clearInterests(profile, store, reader)
		case "s":
			saveChanges(profile, store)
		case "x", "q":
			saveChanges(profile, store)
			return
		default:
			fmt.Println(errorStyle.Render("\nInvalid option. Please try again."))
			time.Sleep(1 * time.Second)
		}
	}
}

// Helper functions to break down the functionality
func addInterest(profile *models.UserProfile, store *storage.Storage, reader *bufio.Reader) {
	fmt.Print(promptStyle.Render("\nEnter new interest keyword: "))
	keyword, _ := reader.ReadString('\n')
	keyword = strings.TrimSpace(strings.ToLower(keyword))

	if keyword == "" {
		fmt.Println(errorStyle.Render("\nInterest cannot be empty!"))
		time.Sleep(1 * time.Second)
		return
	}

	fmt.Print(promptStyle.Render("Enter weight (0.1-5.0): "))
	weightStr, _ := reader.ReadString('\n')
	weight, err := strconv.ParseFloat(strings.TrimSpace(weightStr), 64)
	if err != nil {
		fmt.Println(errorStyle.Render("\nInvalid weight value!"))
		time.Sleep(1 * time.Second)
		return
	}

	weight = math.Max(0.1, math.Min(5.0, weight))
	profile.Interests[keyword] = weight

	if err := store.SaveProfile(profile); err != nil {
		fmt.Printf("Error saving profile: %v\n", err)
	} else {
		showSpinner("Adding interest...", 500*time.Millisecond)
		fmt.Println(infoStyle.Render("\n✨ Interest added and saved!"))
	}
	time.Sleep(1 * time.Second)
}

func removeInterest(profile *models.UserProfile, store *storage.Storage, reader *bufio.Reader, sorted []weightedWord) {
	if len(sorted) == 0 {
		fmt.Println(errorStyle.Render("\nNo interests to remove!"))
		time.Sleep(1 * time.Second)
		return
	}

	fmt.Print(promptStyle.Render("\nEnter number to remove (or 'c' to cancel): "))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "c" {
		return
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(sorted) {
		fmt.Println(errorStyle.Render("\nInvalid selection!"))
		time.Sleep(1 * time.Second)
		return
	}

	word := sorted[num-1].word
	delete(profile.Interests, word)

	if err := store.SaveProfile(profile); err != nil {
		fmt.Printf("Error saving profile: %v\n", err)
	} else {
		showSpinner("Removing interest...", 500*time.Millisecond)
		fmt.Println(infoStyle.Render("\n✨ Interest removed and saved!"))
	}
	time.Sleep(1 * time.Second)
}

func saveChanges(profile *models.UserProfile, store *storage.Storage) {
	if err := store.SaveProfile(profile); err != nil {
		fmt.Printf("Error saving profile: %v\n", err)
		fmt.Println(errorStyle.Render("\nFailed to save changes!"))
	} else {
		showSpinner("Saving changes...", 500*time.Millisecond)
		fmt.Println(infoStyle.Render("\n✨ Changes saved successfully!"))
	}
	time.Sleep(1 * time.Second)
}

func clearInterests(profile *models.UserProfile, store *storage.Storage, reader *bufio.Reader) {
	fmt.Print(promptStyle.Render("\nAre you sure you want to clear all interests? (y/n): "))
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) == "y" {
		profile.Interests = make(map[string]float64)

		// Save after clearing interests
		if err := store.SaveProfile(profile); err != nil {
			fmt.Printf("Error saving profile: %v\n", err)
		} else {
			showSpinner("Clearing interests...", 500*time.Millisecond)
			fmt.Println(infoStyle.Render("\n✨ All interests cleared and saved!"))
		}
		time.Sleep(1 * time.Second)
	}
}

func modifyInterest(profile *models.UserProfile, store *storage.Storage, reader *bufio.Reader, sorted []weightedWord) {
	if len(sorted) == 0 {
		fmt.Println(errorStyle.Render("\nNo interests to modify!"))
		time.Sleep(1 * time.Second)
		return
	}

	fmt.Print(promptStyle.Render("\nEnter number to modify: "))
	numStr, _ := reader.ReadString('\n')
	if num, err := strconv.Atoi(strings.TrimSpace(numStr)); err == nil && num > 0 && num <= len(sorted) {
		word := sorted[num-1].word
		fmt.Print(promptStyle.Render("Enter new weight (0.1-5.0): "))
		weightStr, _ := reader.ReadString('\n')
		if weight, err := strconv.ParseFloat(strings.TrimSpace(weightStr), 64); err == nil {
			weight = math.Max(0.1, math.Min(5.0, weight))
			profile.Interests[word] = weight
			showSpinner("Updating weight...", 500*time.Millisecond)
			fmt.Println(infoStyle.Render("\n✨ Weight updated!"))
			time.Sleep(1 * time.Second)
		}
	}
}

func displayResults(items []FeedItem, profile *models.UserProfile, store *storage.Storage) {
	for _, item := range items {
		displayArticle(item, profile, store)
	}
}

func displayArticle(item FeedItem, profile *models.UserProfile, store *storage.Storage) {
	fmt.Println(dividerStyle.Render())

	fmt.Printf("%s %s\n",
		subtitleStyle.Render("📰 Source:"),
		textStyle.Render(item.FeedSource))

	fmt.Printf("%s %s\n",
		subtitleStyle.Render("📌 Title:"),
		titleStyle.Render(item.Title))

	fmt.Printf("%s %s\n",
		subtitleStyle.Render("🕒 Published:"),
		textStyle.Render(item.Published))

	fmt.Printf("%s %s\n",
		subtitleStyle.Render("🔗 Link:"),
		linkStyle.Render(item.Link))

	fmt.Printf("%s %s\n",
		subtitleStyle.Render("📝 Description:"),
		textStyle.Render(item.Description))

	fmt.Println(dividerStyle.Render())

	fmt.Print(promptStyle.Render("\nMark as interesting? (y/n/q): "))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "q" {
		return
	}

	if response == "y" {
		profile.UpdateInterests(item.Title + " " + item.Description)
		profile.ReadArticles[item.Link] = true

		// Save profile after updating
		if err := store.SaveProfile(profile); err != nil {
			fmt.Printf("Error saving profile: %v\n", err)
		} else {
			fmt.Println(titleStyle.Render("✨ Added to your interests!"))
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

func debugProfile(profile *models.UserProfile) {
	fmt.Println("\nDebug: Current Profile State")
	fmt.Printf("Number of interests: %d\n", len(profile.Interests))
	fmt.Printf("Number of read articles: %d\n", len(profile.ReadArticles))
	fmt.Printf("Last updated: %v\n", profile.LastUpdated)

	if len(profile.Interests) > 0 {
		fmt.Println("\nInterests:")
		for word, weight := range profile.Interests {
			fmt.Printf("- %s: %.2f\n", word, weight)
		}
	}
	fmt.Println()
}

func manageFeeds(store *storage.Storage) {
	for {
		clearScreen()

		feeds, err := store.LoadFeeds()
		if err != nil {
			fmt.Printf("Error loading feeds: %v\n", err)
			return
		}

		header := []string{
			"📑 Feed Management",
			"───────────────",
			"",
			"Current feeds:",
			"",
		}

		fmt.Println(appStyle.Render(strings.Join(header, "\n")))

		for i, feed := range feeds {
			fmt.Printf("%d. %s\n", i+1, feed)
		}

		options := []string{
			"",
			"Options:",
			"  [a] Add new feed",
			"  [r] Remove feed",
			"  [x] Return to main menu",
			"",
			"Enter option [a/r/x]:",
		}

		fmt.Println(appStyle.Render(strings.Join(options, "\n")))
		fmt.Print(searchPromptStyle.Render("→ "))

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(strings.ToLower(choice))

		switch choice {
		case "a":
			fmt.Print(promptStyle.Render("\nEnter feed URL: "))
			url, _ := reader.ReadString('\n')
			url = strings.TrimSpace(url)
			if url != "" {
				feeds = append(feeds, url)
				if err := store.SaveFeeds(feeds); err != nil {
					fmt.Printf("Error saving feeds: %v\n", err)
				} else {
					fmt.Println(infoStyle.Render("\n✨ Feed added successfully!"))
				}
				time.Sleep(1 * time.Second)
			}

		case "r":
			fmt.Print(promptStyle.Render("\nEnter number to remove: "))
			numStr, _ := reader.ReadString('\n')
			if num, err := strconv.Atoi(strings.TrimSpace(numStr)); err == nil && num > 0 && num <= len(feeds) {
				feeds = append(feeds[:num-1], feeds[num:]...)
				if err := store.SaveFeeds(feeds); err != nil {
					fmt.Printf("Error saving feeds: %v\n", err)
				} else {
					fmt.Println(infoStyle.Render("\n✨ Feed removed successfully!"))
				}
				time.Sleep(1 * time.Second)
			}

		case "x":
			return
		}
	}
}
