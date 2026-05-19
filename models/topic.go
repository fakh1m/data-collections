package models

import "time"

type Topic struct {
	ID          int       `json:"id"`
	MainKeyword string    `json:"main_keyword"`
	Alias       string    `json:"alias"`
	CreatedAt   time.Time `json:"created_at"`
}

type SearchResult struct {
	ID        int       `json:"id"`
	TopicID   int       `json:"topic_id"`
	Title     string    `json:"title"`
	Snippet   string    `json:"snippet"`
	URL       string    `json:"url"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
}
