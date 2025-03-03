package main

import (
	"fmt"
	"strings"

	"github.com/mmcdole/gofeed"
)

func main() {
	// Initialize the parser
	fp := gofeed.NewParser()

	// Parse the LessNews RSS feed
	feed, err := fp.ParseURL("https://lessnews.dev/rss.xml")
	if err != nil {
		fmt.Println("Error parsing feed:", err)
		return
	}

	// Print feed information
	fmt.Printf("Feed Title: %s\n", feed.Title)
	fmt.Printf("Feed Description: %s\n", feed.Description)
	fmt.Printf("Number of items: %d\n\n", len(feed.Items))

	// Simple search function
	searchTerm := "joplin" // Change this to search for different terms
	fmt.Printf("Searching for: %s\n\n", searchTerm)

	// Search through items
	for _, item := range feed.Items {
		if strings.Contains(strings.ToLower(item.Title), strings.ToLower(searchTerm)) ||
			strings.Contains(strings.ToLower(item.Description), strings.ToLower(searchTerm)) {
			fmt.Printf("Title: %s\n", item.Title)
			fmt.Printf("Published: %s\n", item.Published)
			fmt.Printf("Link: %s\n", item.Link)
			fmt.Printf("Description: %s\n\n", item.Description)
		}
	}
}
