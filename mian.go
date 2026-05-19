package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// ================= CORES / MODELS =================
type Topic struct {
	ID          int    `json:"id"`
	MainKeyword string `json:"main_keyword"`
	Alias       string `json:"alias"`
}

type SearchResult struct {
	ID          int    `json:"id"`
	TopicID     int    `json:"topic_id"`
	MainKeyword string `json:"main_keyword"`
	KeywordUsed string `json:"keyword_used"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet"`
	URL         string `json:"url"`
}

// Struct untuk membaca response dari Google API
type GoogleResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		Link    string `json:"link"`
	} `json:"items"`
}

// ================= OOP REPOSITORY =================
type Repository struct {
	Db *sql.DB
}

func (r *Repository) SaveTopic(main, alias string) error {
	_, err := r.Db.Exec("INSERT INTO topics (main_keyword, alias) VALUES (?, ?)", main, alias)
	return err
}

func (r *Repository) GetTopics() ([]Topic, error) {
	rows, err := r.Db.Query("SELECT id, main_keyword, alias FROM topics")
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

func (r *Repository) SaveResult(res SearchResult) error {
	_, err := r.Db.Exec("INSERT INTO search_results (topic_id, keyword_used, title, snippet, url) VALUES (?, ?, ?, ?, ?)",
		res.TopicID, res.KeywordUsed, res.Title, res.Snippet, res.URL)
	return err
}

func (r *Repository) GetResults() ([]SearchResult, error) {
	query := `SELECT r.id, r.topic_id, t.main_keyword, r.keyword_used, r.title, r.snippet, r.url
			  FROM search_results r
			  JOIN topics t ON r.topic_id = t.id
			  ORDER BY r.id DESC`
	rows, err := r.Db.Query(query)
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

// ================= OOP SERVICE (DRONE ENGINE) =================
type DroneService struct {
	Repo           *Repository
	GoogleAPIKey   string
	SearchEngineID string
}

func (s *DroneService) FetchAndAggregate(mainKeyword string) error {
	// 1. Ambil semua baris yang punya main_keyword sama (misal: mencari "PNI" dan "Partai Nasional Indonesia")
	rows, err := s.Repo.Db.Query("SELECT id, main_keyword, alias FROM topics WHERE main_keyword = ?", mainKeyword)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var t Topic
		rows.Scan(&t.ID, &t.MainKeyword, &t.Alias)

		// 2. Cari data ke Google berdasarkan Alias/Singkatan tersebut
		log.Printf("Engine mencari data untuk: %s", t.Alias)
		u := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s",
			s.GoogleAPIKey, s.SearchEngineID, url.QueryEscape(t.Alias))

		resp, err := http.Get(u)
		if err != nil {
			log.Println("Gagal ke Google API:", err)
			continue
		}

		var googleResp GoogleResponse
		json.NewDecoder(resp.Body).Decode(&googleResp)
		resp.Body.Close()

		// 3. Simpan dan satukan hasil data ke DB
		for _, item := range googleResp.Items {
			res := SearchResult{
				TopicID:     t.ID,
				KeywordUsed: t.Alias,
				Title:       item.Title,
				Snippet:     item.Snippet,
				URL:         item.Link,
			}
			s.Repo.SaveResult(res)
		}
	}
	return nil
}

// ================= MAIN FUNCTION & ROUTER =================
func main() {
	// KONEKSI DATABASE (Sesuaikan user, password, dan nama DB lu)
	dsn := "root:ROOT_DB_PASSWORD@tcp(127.0.0.1:3306)/mini_drone_emprit?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Database gagal connect:", err)
	}
	defer db.Close()

	repo := &Repository{Db: db}

	// ISI CREDENTIAL GOOGLE LU DI SINI
	droneService := &DroneService{
		Repo:           repo,
		GoogleAPIKey:   "YOUR_GOOGLE_API_KEY",
		SearchEngineID: "YOUR_SEARCH_ENGINE_ID",
	}

	r := gin.Default()

	// Unescape HTML untuk link/URL di template
	r.SetFuncMap(template.FuncMap{
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
	})
	r.LoadHTMLGlob("templates/*")

	// Route: Halaman Utama
	r.GET("/", func(c *gin.Context) {
		topics, _ := repo.GetTopics()
		results, _ := repo.GetResults()
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Topics":  topics,
			"Results": results,
		})
	})

	// Route: Tambah Keyword Pantauan baru
	r.POST("/add-topic", func(c *gin.Context) {
		mainKeyword := c.PostForm("main_keyword")
		alias := c.PostForm("alias")

		if mainKeyword != "" && alias != "" {
			repo.SaveTopic(mainKeyword, alias)
		}
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	// Route: Trigger Crawling / Pencarian Google (Menyatukan data)
	r.POST("/crawl", func(c *gin.Context) {
		mainKeyword := c.PostForm("main_keyword")
		if mainKeyword != "" {
			err := droneService.FetchAndAggregate(mainKeyword)
			if err != nil {
				log.Println("Error saat crawl:", err)
			}
		}
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	log.Println("Aplikasi jalan di http://localhost:8080")
	r.Run(":8080")
}
