package models

import (
	"time"
)

type FeedItem struct {
	Title       string
	Description string
	Link        string
	Published   time.Time
	FeedSource  string
}

type SearchOptions struct {
	StartDate time.Time
	EndDate   time.Time
	Source    string
}

type ArticleScore struct {
	Item  FeedItem
	Score float64
}

type SearchResult struct {
	Item       FeedItem
	Matches    []string
	MatchCount int
}
