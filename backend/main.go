package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var db *pgxpool.Pool

// CORS middleware
func enableCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// Get an env var or use the specified fallback value
func getEnvOrFallback(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// JSON structs for request and response data
type ShortenRequest struct {
	LongURL string `json:"long_url"`
}
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}
type ExpandRequest struct {
	ShortCode string `json:"short_code"`
}
type ExpandResponse struct {
	LongURL    string `json:"long_url"`
	ClickCount int    `json:"click_count"`
}

// Create a string of length n.
// Make a byte array, generate random bytes, 
// encode to base64 to ensure URL safety,
// and truncate to remove excess base64 padding.
func generateShortCode(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:n]
}

// Check for a valid URL format and normalize it.
func isValidURL(str string) (string, bool) {
	str = strings.TrimSpace(str)

	if str == "" {
		return "", false
	}

	// Add https if absent
	if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		str = "https://" + str
	}

	// Validate url
	_, err := url.ParseRequestURI(str)
	return str, err == nil
}

// Accept a POSTed url, shorten it, add it to the database,
// and return the short url.
func shortenHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	
	if r.Method == http.MethodOptions {
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	normalizedURL, valid := isValidURL(req.LongURL)
	if !valid {
		http.Error(w, "Invalid URL provided", http.StatusBadRequest)
		return
	}

	shortCode := generateShortCode(6)
	
	// Add shortcode to database
	var shortID int
	err = db.QueryRow(
		r.Context(),
		"INSERT INTO short_urls (short_code, click_count) VALUES ($1, 0) RETURNING id",
		shortCode,
	).Scan(&shortID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Add full url to database
	var longID int
	err = db.QueryRow(
		r.Context(),
		"INSERT INTO long_urls (url) VALUES ($1) RETURNING id",
		normalizedURL,
	).Scan(&longID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Link shortcode to full url
	_, err = db.Exec(r.Context(),
		"INSERT INTO url_links (long_url_id, short_url_id) VALUES ($1, $2)",
		longID, shortID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Return the complete short URL to the client
	baseURL := getEnvOrFallback("BASE_URL", "http://localhost:8080")
	resp := ShortenResponse{ShortURL: fmt.Sprintf("%s/%s", baseURL, shortCode)}
	json.NewEncoder(w).Encode(resp)
}

// Accept a POSTed url, find the corresponding url in the db,
// and return the original url and click count.
func expandHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	
	if r.Method == http.MethodOptions {
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExpandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.ShortCode) == "" {
		http.Error(w, "Short code cannot be empty", http.StatusBadRequest)
		return
	}

	// Join original url and click count
	var longURL string
	var clickCount int
	err = db.QueryRow(r.Context(),
		`SELECT l.url, s.click_count
         FROM short_urls s
         JOIN url_links ul ON s.id = ul.short_url_id
         JOIN long_urls l ON l.id = ul.long_url_id
         WHERE s.short_code = $1`,
		req.ShortCode).Scan(&longURL, &clickCount)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	resp := ExpandResponse{LongURL: longURL, ClickCount: clickCount}
	json.NewEncoder(w).Encode(resp)
}

// Reads a short url and redirect the user to the corresponding
// long url. Increments click count.
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	// extract shortCode from url
	shortCode := strings.TrimPrefix(r.URL.Path, "/")
	
	// Prevent favicon lookups
	if shortCode == "" || shortCode == "favicon.ico" {
		http.NotFound(w, r)
		return
	}

	// Retrieve original url via short code
	var longURL string
	err := db.QueryRow(r.Context(),
		`SELECT l.url
         FROM short_urls s
         JOIN url_links ul ON s.id = ul.short_url_id
         JOIN long_urls l ON l.id = ul.long_url_id
         WHERE s.short_code = $1`,
		shortCode).Scan(&longURL)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Increment click count
	db.Exec(r.Context(), `UPDATE short_urls SET click_count = click_count + 1 WHERE short_code = $1`, shortCode)
	
	// Redirect to original URL using 301 (as per bitly and tinyurl)
	// https://stackoverflow.com/questions/6221632/when-creating-a-short-url-service-is-302-temporary-redirect-the-best-way-to-go
	http.Redirect(w, r, longURL, http.StatusMovedPermanently)
}


func main() {
	// Load .env file if it exists (for local development)
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("falling back to env vars")
	}

	dbURL := os.Getenv("DATABASE_URL")
	
	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/api/shorten", shortenHandler)
	http.HandleFunc("/api/expand", expandHandler)
	http.HandleFunc("/", redirectHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}