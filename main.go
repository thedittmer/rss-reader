package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"encoding/xml"
	"io"
	"net/http"
	"net/url"

	"github.com/mmcdole/gofeed"
	"github.com/thedittmer/rss-reader/internal/models"
	"github.com/thedittmer/rss-reader/internal/storage"
	"github.com/thedittmer/rss-reader/internal/ui"
	"golang.org/x/term"
)

// Constants
const (
	SortByScore = iota
	SortByDate
	Version = "1.0.0"
)

// Types
type App struct {
	store   *storage.Storage
	profile *models.UserProfile
	feeds   []string
	items   []models.FeedItem
}

type keyPress struct {
	key  byte
	char rune
}

// Main function and initialization
func main() {
	// Initialize signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

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

	// Initialize app
	app := NewApp(store)
	app.Run()
}

func NewApp(store *storage.Storage) *App {
	profile, err := store.LoadProfile()
	if err != nil {
		log.Fatalf("Failed to load profile: %v", err)
	}

	feeds, err := store.LoadFeeds()
	if err != nil {
		log.Printf("Error loading feeds: %v", err)
		feeds = []string{"https://lessnews.dev/rss.xml"}
	}

	return &App{
		store:   store,
		profile: profile,
		feeds:   feeds,
	}
}

func (a *App) Run() {
	// Initial feed refresh
	a.refreshFeeds()

	for {
		a.showMainMenu()
	}
}

func (a *App) showMainMenu() {
	clearScreen()
	fmt.Printf("%s v%s\n", ui.HeaderStyle.Render("RSS Reader"), Version)

	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Commands:")
	fmt.Println(ui.ArrowStyle.Render())
	fmt.Printf("%s (s)earch       Search articles\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (r)ecommended  View recommended articles\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (i)nterests    Manage your interests\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (f)eeds        Manage RSS feeds\n", ui.ArrowStyle.Render())
	fmt.Printf("%s refre(x)h      Update all feeds\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (q)uit         Exit the application\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (h)elp         Show help\n", ui.ArrowStyle.Render())
	fmt.Println()

	fmt.Print(ui.CommandStyle.Render("→ "))

	cmd := readLine()
	a.handleCommand(cmd)
}

func (a *App) handleCommand(cmd string) {
	cmd = strings.ToLower(strings.TrimSpace(cmd))

	switch cmd {
	case "h", "help":
		a.showMainHelp()
		return
	case "s", "search":
		a.searchArticles()
		return
	case "r", "recommended":
		a.showRecommendations()
		return
	case "i", "interests":
		a.manageInterests()
		return
	case "f", "feeds":
		a.manageFeeds()
		return
	case "x", "refresh":
		if !confirmAction("Are you sure you want to refresh all feeds? This may take a while.") {
			fmt.Println(ui.DimStyle.Render("Operation cancelled"))
			return
		}
		a.refreshFeeds()
		return
	case "q", "quit", "exit":
		os.Exit(0)
	default:
		showError("Unknown command")
		return
	}
}

func (a *App) refreshFeeds() {
	stop := showProgress("Updating feeds")
	defer stop()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var items []models.FeedItem

	for _, feedURL := range a.feeds {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			feed := parseFeed(url)
			mu.Lock()
			items = append(items, feed...)
			mu.Unlock()
		}(feedURL)
	}
	wg.Wait()

	a.items = items
	showSuccess("Feeds updated successfully")
}

func (a *App) searchArticles() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Search Articles"))
	fmt.Println()

	fmt.Print(ui.CommandStyle.Render("Enter search term (or 'b' to go back): "))
	query := readLine()

	if strings.ToLower(query) == "b" {
		return
	}

	if query == "" {
		showError("Search term cannot be empty")
		return
	}

	stop := showProgress("Searching articles")
	results := a.searchItems(query)
	stop()

	if len(results) == 0 {
		showError("No articles found")
		return
	}

	a.showSearchResults(query, results)
}

func (a *App) searchItems(query string) []models.FeedItem {
	query = strings.ToLower(query)
	var results []models.FeedItem

	for _, item := range a.items {
		if strings.Contains(strings.ToLower(item.Title), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			results = append(results, item)
		}
	}

	return results
}

func (a *App) showSearchResults(query string, results []models.FeedItem) {
	currentPage := 0
	itemsPerPage := 10
	totalPages := (len(results) + itemsPerPage - 1) / itemsPerPage
	selectedItem := 0

	for {
		clearScreen()
		fmt.Printf("%s Search Results for \"%s\"\n", ui.HeaderStyle.Render("→"), query)
		fmt.Printf("%s Found %d articles\n", ui.DimStyle.Render("→"), len(results))
		fmt.Println()

		// Display results for current page
		start := currentPage * itemsPerPage
		end := min(start+itemsPerPage, len(results))

		for i, item := range results[start:end] {
			cursor := ui.UnselectedStyle.Render()
			if i == selectedItem {
				cursor = ui.SelectedStyle.Render()
			}
			fmt.Printf("%s %s. %s\n",
				cursor,
				ui.DimStyle.Render(fmt.Sprintf("%d", start+i+1)),
				ui.TitleStyle.Render(item.Title))
			fmt.Printf("   %s - %s\n",
				ui.SourceStyle.Render(item.FeedSource),
				ui.DateStyle.Render(item.Published.Format("2006-01-02")))
			fmt.Printf("   %s %s\n",
				ui.DimStyle.Render("Link:"),
				ui.LinkStyle.Render(item.Link))
			fmt.Println()
		}

		// Show navigation help
		fmt.Println()
		fmt.Println(ui.DimStyle.Render("Navigation:"))
		fmt.Printf("%s ↑/↓ or j/k    Navigate items\n", ui.ArrowStyle.Render())
		fmt.Printf("%s ←/→ or h/l    Change pages\n", ui.ArrowStyle.Render())
		fmt.Printf("%s Enter         View selected article\n", ui.ArrowStyle.Render())
		fmt.Printf("%s o             Open in browser\n", ui.ArrowStyle.Render())
		fmt.Printf("%s b             Back to main menu\n", ui.ArrowStyle.Render())
		fmt.Println()

		// Read key input
		key, err := readKey()
		if err != nil {
			continue
		}

		// Handle 'o' followed by number
		if key.key == 'o' {
			var numStr string
			fmt.Print("o") // Show the 'o' being typed

			// Read subsequent digits
			for {
				k, err := readKey()
				if err != nil {
					break
				}
				// If Enter is pressed or non-digit/non-backspace, break
				if k.key == 13 || (k.key != 127 && (k.key < '0' || k.key > '9')) {
					break
				}
				// If backspace, remove last digit
				if k.key == 127 && len(numStr) > 0 {
					numStr = numStr[:len(numStr)-1]
					fmt.Print("\b \b") // Erase character
					continue
				}
				// Add digit and show it
				numStr += string(k.char)
				fmt.Print(string(k.char))
			}
			fmt.Println() // New line after input

			// Process the number
			if numStr != "" {
				if num, err := strconv.Atoi(numStr); err == nil {
					index := num - 1
					if index >= 0 && index < len(results) {
						if err := openInBrowser(results[index].Link); err != nil {
							showError("Failed to open browser")
						} else {
							showSuccess(fmt.Sprintf("Opened article %d in browser", num))
						}
					} else {
						showError(fmt.Sprintf("Invalid article number: %d", num))
					}
				}
			}
			continue
		}

		switch key.key {
		case 'j', 66: // Down arrow
			if selectedItem < min(itemsPerPage-1, end-start-1) {
				selectedItem++
			}
		case 'k', 65: // Up arrow
			if selectedItem > 0 {
				selectedItem--
			}
		case 'l', 67: // Right arrow
			if currentPage < totalPages-1 {
				currentPage++
				selectedItem = 0
			}
		case 68, 'h': // Left arrow or 'h' for left
			if key.key == 'h' {
				a.showSearchHelp()
				continue
			}
			// Otherwise handle as left arrow
			if currentPage > 0 {
				currentPage--
				selectedItem = 0
			}
		case 13: // Enter
			itemIndex := start + selectedItem
			if itemIndex < len(results) {
				a.viewArticleSequence(results, itemIndex)
			}
		case 'b':
			return
		}
	}
}

func (a *App) viewArticleSequence(items []models.FeedItem, startIndex int) {
	for i := startIndex; i < len(items); i++ {
		clearScreen()
		fmt.Printf("%s Article %d of %d\n", ui.DimStyle.Render("→"), i+1, len(items))
		fmt.Println()

		continueViewing := a.displayArticle(items[i])
		if !continueViewing {
			return // Return to results view
		}
	}

	// Show end of results message
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("End of Results"))
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return to results..."))
	readLine()
}

func (a *App) showRecommendations() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Recommended Articles"))
	fmt.Println()

	if len(a.profile.Interests) == 0 {
		showError("No interests set. Add some interests first!")
		return
	}

	// Calculate recommendations
	var recommendations []models.ArticleScore
	for _, item := range a.items {
		score := a.calculateInterestScore(item)
		if score > 0 {
			recommendations = append(recommendations, models.ArticleScore{
				Item:  item,
				Score: score,
			})
		}
	}

	if len(recommendations) == 0 {
		showError("No recommendations found")
		return
	}

	// Sort by score
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Show paginated recommendations
	currentPage := 0
	itemsPerPage := 10
	selectedItem := 0

	// Show sorting options
	fmt.Printf("%s Sorting by: %s\n",
		ui.ArrowStyle.Render(),
		"Relevance")
	fmt.Println()

	for {
		clearScreen()
		fmt.Println(ui.HeaderStyle.Render("Recommended Articles"))
		fmt.Printf("%s Found %d recommendations\n", ui.DimStyle.Render("→"), len(recommendations))
		fmt.Println()

		// Sort articles
		sorted := a.sortRecommendations(recommendations, SortByScore)

		// Calculate pagination
		totalPages := (len(sorted) + itemsPerPage - 1) / itemsPerPage
		start := currentPage * itemsPerPage
		end := min(start+itemsPerPage, len(sorted))

		// Show articles
		for i, article := range sorted[start:end] {
			cursor := ui.UnselectedStyle.Render()
			if i == selectedItem {
				cursor = ui.SelectedStyle.Render()
			}
			fmt.Printf("%s %s. %s\n",
				cursor,
				ui.DimStyle.Render(fmt.Sprintf("%d", start+i+1)),
				ui.TitleStyle.Render(article.Item.Title))
			fmt.Printf("   %s - %s\n",
				ui.SourceStyle.Render(article.Item.FeedSource),
				ui.DateStyle.Render(article.Item.Published.Format("2006-01-02")))
			fmt.Printf("   %s %.2f\n",
				ui.DimStyle.Render("Score:"),
				ui.ScoreStyle.Render(fmt.Sprintf("%.2f", article.Score)))
			fmt.Println()
		}

		// Show pagination info
		fmt.Printf("%s Page %d of %d (%d articles)\n",
			ui.ArrowStyle.Render(),
			currentPage+1,
			totalPages,
			len(sorted))
		fmt.Println()

		// Show commands
		fmt.Println(ui.ArrowStyle.Render() + "Commands:")
		fmt.Printf("%s (n)ext/(p)rev    Navigate pages\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (s)ort           Toggle sort (relevance/date)\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (v)iew [number]  View article details\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (o)[number]      Open in browser\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (b)ack           Return to main menu\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (h)elp           Show help\n", ui.ArrowStyle.Render())
		fmt.Println()

		// Add keyboard navigation
		key, err := readKey()
		if err != nil {
			continue
		}

		// Handle 'o' followed by number
		if key.key == 'o' {
			var numStr string
			fmt.Print("o") // Show the 'o' being typed

			// Read subsequent digits
			for {
				k, err := readKey()
				if err != nil {
					break
				}
				// If Enter is pressed or non-digit/non-backspace, break
				if k.key == 13 || (k.key != 127 && (k.key < '0' || k.key > '9')) {
					break
				}
				// If backspace, remove last digit
				if k.key == 127 && len(numStr) > 0 {
					numStr = numStr[:len(numStr)-1]
					fmt.Print("\b \b") // Erase character
					continue
				}
				// Add digit and show it
				numStr += string(k.char)
				fmt.Print(string(k.char))
			}
			fmt.Println() // New line after input

			// Process the number
			if numStr != "" {
				if num, err := strconv.Atoi(numStr); err == nil {
					index := num - 1
					if index >= 0 && index < len(sorted) {
						if err := openInBrowser(sorted[index].Item.Link); err != nil {
							showError("Failed to open browser")
						} else {
							showSuccess(fmt.Sprintf("Opened article %d in browser", num))
						}
					} else {
						showError(fmt.Sprintf("Invalid article number: %d", num))
					}
				}
			}
			continue
		}

		// Handle navigation similar to showSearchResults
		switch key.key {
		case 'j', 66: // Down arrow
			if selectedItem < min(itemsPerPage-1, end-start-1) {
				selectedItem++
			}
		case 'k', 65: // Up arrow
			if selectedItem > 0 {
				selectedItem--
			}
		case 'l', 67: // Right arrow, next page
			if currentPage < totalPages-1 {
				currentPage++
				selectedItem = 0
			} else {
				showError("Already on last page")
			}
		case 68: // Left arrow
			if currentPage > 0 {
				currentPage--
				selectedItem = 0
			}
		case 'h': // Help - separate case for help command
			a.showRecommendationsHelp()
			continue
		case 'n': // Next page
			if currentPage < totalPages-1 {
				currentPage++
				selectedItem = 0
			} else {
				showError("Already on last page")
			}
		case 'p': // Previous page
			if currentPage > 0 {
				currentPage--
				selectedItem = 0
			} else {
				showError("Already on first page")
			}
		case 's': // Sort
			if currentPage == 0 {
				currentPage = totalPages - 1
			} else {
				currentPage = 0
			}
		case 'v': // View
			itemIndex := start + selectedItem
			if itemIndex < len(sorted) {
				// Convert ArticleScore slice to FeedItem slice
				items := make([]models.FeedItem, len(sorted))
				for i, score := range sorted {
					items[i] = score.Item
				}
				a.viewArticleSequence(items, itemIndex)
			}
		case 'b': // Back
			return
		case 13: // Enter
			itemIndex := start + selectedItem
			if itemIndex < len(sorted) {
				// Convert ArticleScore slice to FeedItem slice
				items := make([]models.FeedItem, len(sorted))
				for i, score := range sorted {
					items[i] = score.Item
				}
				a.viewArticleSequence(items, itemIndex)
			}
		}
	}
}

// Add this helper function for sorting articles
func (a *App) sortRecommendations(articles []models.ArticleScore, sortBy int) []models.ArticleScore {
	sorted := make([]models.ArticleScore, len(articles))
	copy(sorted, articles)

	switch sortBy {
	case SortByScore:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Score > sorted[j].Score // Higher scores first
		})
	case SortByDate:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Item.Published.After(sorted[j].Item.Published) // Newer first
		})
	}
	return sorted
}

// Add a help function for recommendations
func (a *App) showRecommendationsHelp() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Help - Recommendations"))
	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Available Commands:")
	fmt.Println()
	fmt.Printf("%s next (n)          Go to next page\n", ui.ArrowStyle.Render())
	fmt.Printf("%s prev (p)          Go to previous page\n", ui.ArrowStyle.Render())
	fmt.Printf("%s sort (s)          Toggle between relevance and date sorting\n", ui.ArrowStyle.Render())
	fmt.Printf("%s view (v) [num]    View article details\n", ui.ArrowStyle.Render())
	fmt.Printf("%s o[num]            Open article in browser\n", ui.ArrowStyle.Render())
	fmt.Printf("%s back (b)          Return to main menu\n", ui.ArrowStyle.Render())
	fmt.Printf("%s help (h)          Show this help message\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Relevance sorting shows articles based on your interests\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Date sorting shows newest articles first\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Use numbers to quickly view specific articles\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return..."))
	readLine()
}

func (a *App) calculateInterestScore(item models.FeedItem) float64 {
	var score float64
	text := strings.ToLower(item.Title + " " + item.Description)

	for word, weight := range a.profile.Interests {
		if strings.Contains(text, strings.ToLower(word)) {
			score += weight
		}
	}

	return score
}

func (a *App) manageInterests() {
	for {
		clearScreen()
		fmt.Println(ui.HeaderStyle.Render("Manage Interests"))
		fmt.Println()

		if len(a.profile.Interests) == 0 {
			fmt.Println(ui.DimStyle.Render("No interests set"))
		} else {
			for word, weight := range a.profile.Interests {
				fmt.Printf("%s %s (%.2f)\n", ui.ArrowStyle.Render(), word, weight)
			}
		}

		fmt.Println()
		fmt.Println(ui.ArrowStyle.Render() + "Commands:")
		fmt.Printf("%s (a)dd     Add new interest\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (s)core   Set interest weight\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (r)emove  Remove interest\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (b)ack    Return to main menu\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (h)elp    Show help\n", ui.ArrowStyle.Render())
		fmt.Println()

		fmt.Print(ui.CommandStyle.Render("→ "))
		cmd := readLine()

		switch strings.ToLower(cmd) {
		case "a", "add":
			fmt.Print(ui.CommandStyle.Render("Enter interest: "))
			interest := readLine()
			if interest != "" {
				fmt.Print(ui.CommandStyle.Render("Enter weight (0.1-10.0): "))
				weightStr := readLine()
				weight, err := strconv.ParseFloat(weightStr, 64)
				if err != nil || weight < 0.1 || weight > 10.0 {
					showError("Invalid weight. Using default weight of 1.0")
					weight = 1.0
				}
				a.profile.Interests[interest] = weight
				if err := a.store.SaveProfile(a.profile); err != nil {
					showError("Failed to save profile")
				} else {
					showSuccess("Interest added")
				}
			}
		case "s", "score":
			if len(a.profile.Interests) == 0 {
				showError("No interests to score")
				continue
			}

			fmt.Println()
			fmt.Println(ui.ArrowStyle.Render() + "Current interests:")
			interests := make([]string, 0, len(a.profile.Interests))
			for interest := range a.profile.Interests {
				interests = append(interests, interest)
			}
			sort.Strings(interests)

			for i, interest := range interests {
				weight := a.profile.Interests[interest]
				fmt.Printf("%s %d. %s (weight: %.2f)\n", ui.ArrowStyle.Render(), i+1, interest, weight)
			}

			fmt.Println()
			fmt.Print(ui.CommandStyle.Render("Enter interest number to score: "))
			input := readLine()

			index, err := strconv.Atoi(input)
			if err != nil || index < 1 || index > len(interests) {
				showError("Invalid interest number")
				continue
			}

			interest := interests[index-1]
			fmt.Printf(ui.CommandStyle.Render("Enter new weight for '%s' (0.1-10.0): "), interest)
			weightStr := readLine()
			weight, err := strconv.ParseFloat(weightStr, 64)
			if err != nil || weight < 0.1 || weight > 10.0 {
				showError("Invalid weight")
				continue
			}

			a.profile.Interests[interest] = weight
			if err := a.store.SaveProfile(a.profile); err != nil {
				showError("Failed to save profile: " + err.Error())
				continue
			}

			showSuccess("Interest weight updated")
			continue
		case "r", "remove":
			if len(a.profile.Interests) == 0 {
				showError("No interests to remove")
				continue
			}

			fmt.Println()
			fmt.Println(ui.ArrowStyle.Render() + "Current interests:")
			interests := make([]string, 0, len(a.profile.Interests))
			for interest := range a.profile.Interests {
				interests = append(interests, interest)
			}
			sort.Strings(interests)

			for i, interest := range interests {
				weight := a.profile.Interests[interest]
				fmt.Printf("%s %d. %s (weight: %.2f)\n", ui.ArrowStyle.Render(), i+1, interest, weight)
			}

			fmt.Println()
			fmt.Print(ui.CommandStyle.Render("Enter interest number to remove: "))
			input := readLine()

			index, err := strconv.Atoi(input)
			if err != nil || index < 1 || index > len(interests) {
				showError("Invalid interest number")
				continue
			}

			interest := interests[index-1]
			if !confirmAction(fmt.Sprintf("Are you sure you want to remove '%s'?", interest)) {
				fmt.Println(ui.DimStyle.Render("Operation cancelled"))
				continue
			}

			// Remove the interest
			delete(a.profile.Interests, interest)
			if err := a.store.SaveProfile(a.profile); err != nil {
				showError("Failed to save profile: " + err.Error())
				continue
			}

			fmt.Println(ui.SuccessStyle.Render("Interest removed successfully"))
			continue
		case "b", "back":
			return
		case "h", "help":
			a.showInterestsHelp()
			continue
		default:
			showError("Unknown command")
		}
	}
}

func (a *App) manageFeeds() {
	for {
		clearScreen()
		fmt.Println(ui.HeaderStyle.Render("Manage Feeds"))
		fmt.Println()

		for i, feed := range a.feeds {
			fmt.Printf("%s %d. %s\n", ui.ArrowStyle.Render(), i+1, feed)
		}

		fmt.Println()
		fmt.Println(ui.ArrowStyle.Render() + "Commands:")
		fmt.Printf("%s (a)dd     Add new feed\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (r)emove  Remove feed\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (b)ack    Return to main menu\n", ui.ArrowStyle.Render())
		fmt.Printf("%s (h)elp    Show help\n", ui.ArrowStyle.Render())
		fmt.Println()

		fmt.Print(ui.CommandStyle.Render("→ "))
		cmd := readLine()

		switch strings.ToLower(cmd) {
		case "h", "help":
			a.showFeedsHelp()
			continue
		case "a", "add":
			fmt.Println()
			fmt.Print(ui.CommandStyle.Render("Enter feed URL: "))
			feedURL := strings.TrimSpace(readLine())

			// Normalize and validate URL format
			normalizedURL, err := normalizeURL(feedURL)
			if err != nil {
				showError(err.Error())
				continue
			}

			// Check if feed already exists
			for _, existingURL := range a.feeds {
				if existingURL == normalizedURL {
					showError("This feed is already in your list")
					continue
				}
			}

			// Show validation progress
			fmt.Print(ui.DimStyle.Render("Validating feed... "))

			// Validate feed content
			if err := a.validateFeed(normalizedURL); err != nil {
				fmt.Println(ui.ErrorStyle.Render("Failed"))
				showError(err.Error())
				continue
			}
			fmt.Println(ui.SuccessStyle.Render("OK"))

			// Confirm adding the feed
			if !confirmAction(fmt.Sprintf("Add %s to your list?", normalizedURL)) {
				fmt.Println(ui.DimStyle.Render("Operation cancelled"))
				continue
			}

			// Add the feed
			a.feeds = append(a.feeds, normalizedURL)
			if err := a.store.SaveFeeds(a.feeds); err != nil {
				showError("Failed to save feeds: " + err.Error())
				continue
			}

			fmt.Println(ui.SuccessStyle.Render("Feed added successfully"))

			// Offer to refresh feeds
			if confirmAction("Would you like to refresh feeds now to fetch articles?") {
				a.refreshFeeds()
			}
			continue
		case "r", "remove":
			if len(a.feeds) == 0 {
				showError("No feeds to remove")
				continue
			}

			fmt.Println()
			fmt.Println(ui.ArrowStyle.Render() + "Current feeds:")
			for i, feed := range a.feeds {
				fmt.Printf("%s %d. %s\n", ui.ArrowStyle.Render(), i+1, feed)
			}

			fmt.Println()
			fmt.Print(ui.CommandStyle.Render("Enter feed number to remove: "))
			input := readLine()

			index, err := strconv.Atoi(input)
			if err != nil || index < 1 || index > len(a.feeds) {
				showError("Invalid feed number")
				continue
			}

			feedURL := a.feeds[index-1]
			if !confirmAction(fmt.Sprintf("Are you sure you want to remove '%s'?", feedURL)) {
				fmt.Println(ui.DimStyle.Render("Operation cancelled"))
				continue
			}

			// Remove the feed
			a.feeds = append(a.feeds[:index-1], a.feeds[index:]...)
			if err := a.store.SaveFeeds(a.feeds); err != nil {
				showError("Failed to save feeds: " + err.Error())
				continue
			}

			fmt.Println(ui.SuccessStyle.Render("Feed removed successfully"))
			continue
		case "b", "back":
			return
		default:
			showError("Unknown command")
		}
	}
}

func (a *App) displayResults(items []models.FeedItem) {
	for i, item := range items {
		clearScreen()
		fmt.Printf("%s Article %d of %d\n", ui.DimStyle.Render("→"), i+1, len(items))
		fmt.Println()

		continueViewing := a.displayArticle(item)
		if !continueViewing {
			return // Only return to main menu if user chooses 'back'
		}
	}

	// Show end of results message
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("End of Results"))
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return to main menu..."))
	readLine()
}

func (a *App) displayArticle(item models.FeedItem) bool {
	clearScreen()
	fmt.Println(ui.TitleStyle.Render(item.Title))
	fmt.Printf("%s %s\n",
		ui.DimStyle.Render("Source:"),
		ui.SourceStyle.Render(item.FeedSource))
	fmt.Printf("%s %s\n",
		ui.DimStyle.Render("Published:"),
		ui.DateStyle.Render(item.Published.Format("2006-01-02")))
	fmt.Println()
	fmt.Println(wordWrap(item.Description, 80))
	fmt.Println()
	fmt.Printf("%s %s\n",
		ui.DimStyle.Render("Link:"),
		ui.LinkStyle.Render(item.Link))
	fmt.Println()

	// Show commands with enhanced styling
	fmt.Println(ui.SectionStyle.Render("Commands:"))
	fmt.Printf("%s %s Mark as interesting and continue\n",
		ui.KeyStyle.Render("(y)es"),
		ui.DimStyle.Render("→"))
	fmt.Printf("%s (n)o      Skip to next article\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (b)ack    Return to main menu\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (o)pen    Open in browser\n", ui.ArrowStyle.Render())
	fmt.Printf("%s (h)elp    Show help\n", ui.ArrowStyle.Render())
	fmt.Println()

	// Show tips
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Marking articles as interesting improves recommendations\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Use 'o' to read full article in your browser\n", ui.ArrowStyle.Render())
	fmt.Println()

	// Read and handle command
	fmt.Print(ui.CommandStyle.Render("→ "))
	cmd := readLine()

	switch strings.ToLower(cmd) {
	case "y", "yes":
		// Handle marking as interesting
		return true
	case "n", "no":
		return true
	case "b", "back":
		return false
	case "o", "open":
		if err := openInBrowser(item.Link); err != nil {
			showError("Failed to open browser")
		}
		return true
	case "h", "help":
		return a.showArticleHelp()
	default:
		showError("Unknown command")
		return true
	}
}

func (a *App) showInterestsHelp() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Help - Manage Interests"))
	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Available Commands:")
	fmt.Println()
	fmt.Printf("%s add (a)           Add a new interest\n", ui.ArrowStyle.Render())
	fmt.Printf("%s remove (r)        Remove an existing interest\n", ui.ArrowStyle.Render())
	fmt.Printf("%s back (b)          Return to main menu\n", ui.ArrowStyle.Render())
	fmt.Printf("%s help (h)          Show this help message\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Interests help find articles you'll like\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Interest weights increase as you mark articles\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Higher weights mean stronger recommendations\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return..."))
	readLine()
	return
}

func (a *App) showFeedsHelp() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Help - Manage Feeds"))
	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Available Commands:")
	fmt.Println()
	fmt.Printf("%s add (a)           Add a new RSS feed\n", ui.ArrowStyle.Render())
	fmt.Printf("%s remove (r)        Remove an existing feed\n", ui.ArrowStyle.Render())
	fmt.Printf("%s back (b)          Return to main menu\n", ui.ArrowStyle.Render())
	fmt.Printf("%s help (h)          Show this help message\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Enter the full URL of the RSS feed\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Feeds are automatically updated on startup\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Use refresh (x) in main menu to update manually\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return..."))
	readLine()
	return
}

// Helper functions
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// Add this function to read a single keypress
func readKey() (keyPress, error) {
	// Put terminal into raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return keyPress{}, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var buf [3]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return keyPress{}, err
	}

	// Handle arrow keys and special keys
	if buf[0] == 27 && n == 3 {
		switch buf[2] {
		case 65: // Up arrow
			return keyPress{key: 'k'}, nil
		case 66: // Down arrow
			return keyPress{key: 'j'}, nil
		case 67: // Right arrow
			return keyPress{key: 'l'}, nil
		case 68: // Left arrow
			return keyPress{key: 'h'}, nil
		}
	}

	// Handle regular keys
	if n == 1 {
		return keyPress{key: buf[0], char: rune(buf[0])}, nil
	}

	return keyPress{}, nil
}

// Update the readLine function to handle arrow keys
func readLine() string {
	// Get the original terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return ""
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var line string
	var buf [1024]byte

	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return line
		}

		for i := 0; i < n; i++ {
			// Handle Enter key
			if buf[i] == 13 {
				fmt.Println() // New line after input
				return line
			}

			// Handle backspace/delete
			if buf[i] == 127 || buf[i] == 8 {
				if len(line) > 0 {
					line = line[:len(line)-1]
					fmt.Print("\b \b") // Erase character
				}
				continue
			}

			// Handle printable characters and pasted content
			if buf[i] >= 32 && buf[i] <= 126 {
				line += string(buf[i])
				fmt.Print(string(buf[i]))
			}
		}
	}
}

func showSuccess(msg string) {
	fmt.Printf("%s %s\n",
		ui.SuccessStyle.Render("✓"),
		msg)
	time.Sleep(1 * time.Second)
}

func showError(msg string) {
	fmt.Printf("%s %s\n",
		ui.ErrorStyle.Render("✗"),
		msg)
	time.Sleep(1 * time.Second)
}

func showProgress(msg string) func() {
	done := make(chan bool)
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\r%s %s",
					ui.CommandStyle.Render(frames[i]),
					msg)
				i = (i + 1) % len(frames)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return func() {
		done <- true
		fmt.Println()
	}
}

func openInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func wordWrap(text string, width int) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return ""
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseFeed(url string) []models.FeedItem {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil
	}

	var items []models.FeedItem
	for _, item := range feed.Items {
		// Parse the published date
		published := time.Now() // default to current time
		if item.Published != "" {
			if t, err := parseDate(item.Published); err == nil {
				published = t
			}
		}

		items = append(items, models.FeedItem{
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			Published:   published,
			FeedSource:  feed.Title,
		})
	}
	return items
}

func (a *App) showMainHelp() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Help - Main Menu"))
	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Available Commands:")
	fmt.Println()
	fmt.Printf("%s search (s)       Search through all articles\n", ui.ArrowStyle.Render())
	fmt.Printf("%s recommended (r)   View articles based on your interests\n", ui.ArrowStyle.Render())
	fmt.Printf("%s interests (i)     Add or remove topics you're interested in\n", ui.ArrowStyle.Render())
	fmt.Printf("%s feeds (f)         Manage your RSS feed subscriptions\n", ui.ArrowStyle.Render())
	fmt.Printf("%s refresh (x)       Update all feeds to get latest articles\n", ui.ArrowStyle.Render())
	fmt.Printf("%s quit (q)          Exit the application\n", ui.ArrowStyle.Render())
	fmt.Printf("%s help (h)          Show this help message\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Use single-letter commands for faster navigation\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Your interests affect article recommendations\n", ui.ArrowStyle.Render())
	fmt.Printf("%s All changes are automatically saved\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return..."))
	readLine()
}

func (a *App) showSearchHelp() {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Help - Search Results"))
	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Available Commands:")
	fmt.Println()
	fmt.Printf("%s next (n)          Go to next page\n", ui.ArrowStyle.Render())
	fmt.Printf("%s prev (p)          Go to previous page\n", ui.ArrowStyle.Render())
	fmt.Printf("%s view (v)          View article details\n", ui.ArrowStyle.Render())
	fmt.Printf("%s back (b)          Return to main menu\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Use single-letter commands for faster navigation\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Your interests affect article recommendations\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return..."))
	readLine()
	return
}

func (a *App) showArticleHelp() bool {
	clearScreen()
	fmt.Println(ui.HeaderStyle.Render("Help - Article View"))
	fmt.Println()
	fmt.Println(ui.ArrowStyle.Render() + "Available Commands:")
	fmt.Println()
	fmt.Printf("%s yes (y)           Mark as interesting and continue\n", ui.ArrowStyle.Render())
	fmt.Printf("%s no (n)            Skip to next article\n", ui.ArrowStyle.Render())
	fmt.Printf("%s back (b)          Return to main menu\n", ui.ArrowStyle.Render())
	fmt.Printf("%s open (o)          Open in browser\n", ui.ArrowStyle.Render())
	fmt.Printf("%s help (h)          Show this help message\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Tips:"))
	fmt.Printf("%s Marking articles as interesting improves recommendations\n", ui.ArrowStyle.Render())
	fmt.Printf("%s Use 'o' to read full article in your browser\n", ui.ArrowStyle.Render())
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Press Enter to return..."))
	readLine()
	return true
}

// Add this helper function for confirmations
func confirmAction(prompt string) bool {
	fmt.Println()
	fmt.Printf("%s %s (y/n): ", ui.ArrowStyle.Render(), prompt)
	response := strings.ToLower(strings.TrimSpace(readLine()))
	return response == "y" || response == "yes"
}

// Add these helper functions for feed validation
func normalizeURL(urlStr string) (string, error) {
	// If no scheme is provided, prepend https://
	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Ensure scheme is http or https
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("URL must use http or https protocol")
	}

	return u.String(), nil
}

func (a *App) validateFeed(feedURL string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try to fetch the feed
	resp, err := client.Get(feedURL)
	if err != nil {
		return fmt.Errorf("could not connect to feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feed returned status code %d", resp.StatusCode)
	}

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read feed content: %v", err)
	}

	// Try to parse as RSS
	var rss struct {
		XMLName xml.Name `xml:"rss"`
	}
	if xml.Unmarshal(body, &rss) == nil {
		return nil
	}

	// Try to parse as Atom
	var atom struct {
		XMLName xml.Name `xml:"feed"`
	}
	if xml.Unmarshal(body, &atom) == nil {
		return nil
	}

	return fmt.Errorf("URL does not appear to be a valid RSS or Atom feed")
}

// Add this helper function to parse dates
func parseDate(dateStr string) (time.Time, error) {
	// Try common date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	// Return current time and error if parsing fails
	return time.Now(), fmt.Errorf("could not parse date: %s", dateStr)
}
