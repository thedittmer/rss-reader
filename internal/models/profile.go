package models

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	maxInterests = 100  // Maximum number of interests to track
	minWeight    = 0.1  // Minimum weight to keep an interest
	decayFactor  = 0.95 // How much to decay weights over time
)

type UserProfile struct {
	Interests    map[string]float64
	ReadArticles map[string]bool
	LastUpdated  time.Time
}

func NewUserProfile() *UserProfile {
	return &UserProfile{
		Interests:    make(map[string]float64),
		ReadArticles: make(map[string]bool),
		LastUpdated:  time.Now(),
	}
}

// UpdateInterests updates the profile's interests based on the given text
func (p *UserProfile) UpdateInterests(text string) {
	// Extract important words
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
	words := strings.Fields(strings.ToLower(text))
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
