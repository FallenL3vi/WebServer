package main

import (
	"net/http"
	"fmt"
	"sync/atomic"
	"encoding/json"
	"strings"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/FallenL3vi/WebServer/internal/database"
	"database/sql"
	"os"
	"github.com/google/uuid"
	"time"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
}

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
}

type Post struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}


func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html><body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %v times!</p>
	</body></html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) restMiddlewareMetrics(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "ERROR Forbidden You don't have an access", nil)
	}
	cfg.fileserverHits.Swap(0)
	w.WriteHeader(http.StatusOK)
	err := cfg.dbQueries.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR couldon't delete users", err)
		return
	}
	w.Write([]byte("Hits reset to 0"))
}

func cleanBadWord(text string) string {

	if len(text) == 0{
		fmt.Println("Empty text or word to clean\n")
		return ""
	}

	words := strings.Split(text, " ")

	for index, word := range words {
		var lowerStr string = strings.ToLower(word)
		var asterix string = "****"
		switch lowerStr {
			case "kerfuffle":
				words[index] = asterix
			case "sharbert":
				words[index] = asterix
			case "fornax":
				words[index] = asterix
		}
	}

	var newStr string = strings.Join(words, " ")
	return newStr
}


func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	user, err := cfg.dbQueries.CreateUser(r.Context(), params.Email)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error couldn't create an user", err)
		return
	}

	var returnValue User = User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}

	respondWithJSON(w, 201, returnValue)

}

func(cfg *apiConfig) handleMessage(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters:", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Is too long", nil)
		return
	}
	
	var cleanText string = cleanBadWord(params.Body)

	post, err := cfg.dbQueries.CreatePost(r.Context(), database.CreatePostParams{
		Body: cleanText,
		UserID: params.UserID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR Couldn't create a post", err)
		return
	}

	respondWithJSON(w, 201, Post{
		ID: post.ID,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
		Body: post.Body,
		UserID: post.UserID,
	})


}

func(cfg *apiConfig) handleGetPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := cfg.dbQueries.GetPosts(r.Context())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR Couldn't get posts", err)
		return
	}

	returnPosts := []Post{}

	for _, post := range posts {
		returnPosts = append(returnPosts, Post{
			ID: post.ID,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
			Body: post.Body,
			UserID: post.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, returnPosts)
}

func main() {
	//Load and get enviorment variable DB_URL
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	//Open connection to database
	db, err := sql.Open("postgres", dbURL)


	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries: database.New(db),
		platform: platform,
	}

	mux := http.NewServeMux()
	server := http.Server{}
	server.Addr =":8080"
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))


	})
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	
	mux.HandleFunc("GET /admin/metrics", cfg.getRequests)

	mux.HandleFunc("POST /admin/reset", cfg.restMiddlewareMetrics)

	mux.HandleFunc("POST /api/users",  cfg.handlerUsers)

	mux.HandleFunc("POST /api/chirps", cfg.handleMessage)

	mux.HandleFunc("GET /api/chirps", cfg.handleGetPosts)

	server.Handler = mux

	err = server.ListenAndServe()

	if err != nil {
		fmt.Println("Server error:", err)
	}
}