package repository

import (
	"database/sql"
)

// Struktur Model (Bisa dipisah, tapi ditaruh sini agar ringkas)
type Topic struct {
	ID          int
	MainKeyword string
	Alias       string
}

type SearchResult struct {
	ID          int
	TopicID     int
	MainKeyword string
	KeywordUsed string
	Title       string
	Snippet     string
	URL         string
}

// OOP Interface
type TopicRepository interface {
	SaveTopic(main, alias string) error
	GetTopics() ([]Topic, error)
	SaveResult(res SearchResult) error
	GetResults() ([]SearchResult, error)
	GetDB() *sql.DB
}

// Implementasi Struct Repository
type topicRepository struct {
	db *sql.DB
}

// Constructor OOP untuk membuat instance repository baru
func NewTopicRepository(db *sql.DB) TopicRepository {
	return &topicRepository{db: db}
}

func (r *topicRepository) GetDB() *sql.DB {
	return r.db
}

func (r *topicRepository) SaveTopic(main, alias string) error {
	_, err := r.db.Exec("INSERT INTO topics (main_keyword, alias) VALUES (?, ?)", main, alias)
	return err
}

func (r *topicRepository) GetTopics() ([]Topic, error) {
	rows, err := r.db.Query("SELECT id, main_keyword, alias FROM topics")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var t Topic
		rows.Scan(&t.ID, &t.MainKeyword, &t.Alias)
		topics = append(topics, t)
	}
	return topics, nil
}

func (r *topicRepository) SaveResult(res SearchResult) error {
	_, err := r.db.Exec("INSERT INTO search_results (topic_id, keyword_used, title, snippet, url) VALUES (?, ?, ?, ?, ?)",
		res.TopicID, res.KeywordUsed, res.Title, res.Snippet, res.URL)
	return err
}

func (r *topicRepository) GetResults() ([]SearchResult, error) {
	query := `SELECT r.id, r.topic_id, t.main_keyword, r.keyword_used, r.title, r.snippet, r.url
			  FROM search_results r
			  JOIN topics t ON r.topic_id = t.id
			  ORDER BY r.id DESC`
	rows, err := r.DbQuery(query) // Note helper jika ada error pembacaan method
	rows, err = r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var res SearchResult
		rows.Scan(&res.ID, &res.TopicID, &res.MainKeyword, &res.KeywordUsed, &res.Title, &res.Snippet, &res.URL)
		results = append(results, res)
	}
	return results, nil
}
