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
	"github.com/FallenL3vi/WebServer/internal/auth"
	"database/sql"
	"os"
	"github.com/google/uuid"
	"time"
	"sort"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secretJWT string
	polkaKey string
}

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	Token string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed bool `json:"is_chirpy_red"`
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
		fmt.Println("Empty text or word to clean")
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
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error", err)
		return	
	}

	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: hash,})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error couldn't create an user", err)
		return
	}


	var returnValue User = User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		IsChirpyRed: user.IsChirpyRed,

	}

	respondWithJSON(w, 201, returnValue)

}

func(cfg *apiConfig) handleMessage(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Body string `json:"body"`
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
	

	token, err := auth.GetBearerToken(r.Header)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "ERROR couldn't get token from header", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secretJWT)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "ERROR WRONG JWT ACCESS DENIED", err)
		return
	}

	//ADD VERIFICATION ON DATABASE TO CEHCK IF USER STILL EXISTS
	var cleanText string = cleanBadWord(params.Body)

	post, err := cfg.dbQueries.CreatePost(r.Context(), database.CreatePostParams{
		Body: cleanText,
		UserID: userID,
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
	author_id := r.URL.Query().Get("author_id")
	sortOrder := r.URL.Query().Get("sort")

	var err error

	posts := []database.Post{}


	if author_id != "" {
		authorUUID, err := uuid.Parse(author_id)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "ERROR  couldn't parse string", err)
		}

		posts, err = cfg.dbQueries.GetUserPosts(r.Context(), authorUUID)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "ERROR Couldn't get posts", err)
			return
		}
	} else {
		posts, err = cfg.dbQueries.GetPosts(r.Context())

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "ERROR Couldn't get posts", err)
			return
		}
	}

	returnPosts := []Post{}

	if sortOrder == "desc" {
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].CreatedAt.After(posts[j].CreatedAt)
		})
	}

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

func(cfg *apiConfig) handleGetSinglePost(w http.ResponseWriter, r *http.Request) {
	var postID string = r.PathValue("chirpID")
	if postID == "" {
		respondWithError(w, http.StatusInternalServerError, "ERROR  missing the post ID", nil)
		return
	}

	newUUID, err := uuid.Parse(postID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR  couldn't parse string", err)
		return
	}

	post, err := cfg.dbQueries.GetPost(r.Context(), newUUID)

	if err != nil {
		respondWithError(w, 404, "ERROR  couldn't find the post", err)
		return
	}

	respondWithJSON(w, http.StatusOK, Post{
		ID: post.ID,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
		Body: post.Body,
		UserID: post.UserID,
	})
}

func(cfg *apiConfig) handleLoginUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters:", err)
		return
	}

	var expiresInSeconds int = 3600 //1h

	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), params.Email)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error couldn't get the user", err)
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, params.Password)

	if err != nil {
		respondWithError(w, 401, "ERROR UNAUTHORIZED ACCESS", err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secretJWT, time.Duration(expiresInSeconds)*time.Second)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR couldn't make JWT", err)
		return
	}


	refreshToken, err := auth.MakeRefreshToken()

	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token: refreshToken,
		ExpiresAt: time.Now().UTC().Add(60*24*time.Hour),
		UserID: user.ID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR couldn't save Refresh Token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		Token: token,
		RefreshToken: refreshToken,
		IsChirpyRed: user.IsChirpyRed,

	})

}

func(cfg *apiConfig) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	type responseParams struct {
		Token string `json:"token"`
	}

	tokenHeader, err := auth.GetBearerToken(r.Header)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR couldn't get Refresh Token", err)
		return
	}

	tokenRefresh, err := cfg.dbQueries.GetRefreshToken(r.Context(), tokenHeader)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "ERROR WRONG REFRESH TOKEN ACCESS DENIED", err)
		return
	}

	token, err := auth.MakeJWT(tokenRefresh.UserID, cfg.secretJWT, time.Hour,)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR couldn't make JWT", err)
		return
	}

	respondWithJSON(w, http.StatusOK, responseParams{
		Token: token,
	})

}

func(cfg *apiConfig) handleRefreshRevoke(w http.ResponseWriter, r *http.Request) {
	tokenHeader, err := auth.GetBearerToken(r.Header)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR couldn't get Refresh Token", err)
		return
	}

	err = cfg.dbQueries.SetRevokeAt(r.Context(), tokenHeader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR COULDN'T REVOKE SESSION", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func(cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	accessToken, err := auth.GetBearerToken(r.Header)

	userID, err := auth.ValidateJWT(accessToken, cfg.secretJWT)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "ERROR WRONG JWT ACCESS DENIED", err)
		return
	}


	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	type returnParams struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters:", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error couldn't hash the password", err)
		return
	}

	user, err := cfg.dbQueries.UpdateUserPasswordAndEmail(r.Context(), database.UpdateUserPasswordAndEmailParams{
		HashedPassword: hashedPassword,
		Email: params.Email,
		ID: userID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error couldn't update the user", err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, returnParams{
		ID: userID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		IsChirpyRed: user.IsChirpyRed,
	})
}

func(cfg *apiConfig) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	accessToken, err := auth.GetBearerToken(r.Header)

	userID, err := auth.ValidateJWT(accessToken, cfg.secretJWT)

	if err != nil {
		respondWithError(w, 401, "ERROR WRONG JWT ACCESS DENIED", err)
		return
	}

	var postID string = r.PathValue("chirpID")

	if postID == "" {
		respondWithError(w, http.StatusInternalServerError, "ERROR  missing the post ID", nil)
		return
	}

	PostUUID, err := uuid.Parse(postID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR  couldn't parse string", err)
		return
	}

	post, err := cfg.dbQueries.GetPost(r.Context(), PostUUID)

	if err != nil {
		respondWithError(w, 404, "ERRPR COULDN'T FIND THE POST", err)
	}

	if post.UserID != userID {
		respondWithError(w, 403, "ERROR UNAUTHORIZED ACCESS", nil)
	}

	
	results, err := cfg.dbQueries.DeletePost(r.Context(), database.DeletePostParams {
		ID: PostUUID,
		UserID: userID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR  couldn't delete the post", err)
		return
	}

	rowsAffected, err := results.RowsAffected()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR  checking affected rows", err)
		return
	}

	if rowsAffected == 0 {
		respondWithError(w, 404, "ERROR  POST WAS NOT FOUND OR UNAUTHORIZED", err)
		return
	}
	
	w.WriteHeader(204)
}

func(cfg *apiConfig) handleUpgradeUser(w http.ResponseWriter, r *http.Request) {
	polkaKey, err := auth.GetAPIKey(r.Header)

	if err != nil {
		respondWithError(w, 401, "ERROR Couldn't exxtract API KEY", err)
		return
	}

	if polkaKey  != cfg.polkaKey {
		respondWithError(w, 401, "ERROR AUTHENTICATION FAILED", err)
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters:", err)
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	resultSQL, err := cfg.dbQueries.UpgradeUser(r.Context(), database.UpgradeUserParams{
		IsChirpyRed: true,
		ID: params.Data.UserID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Couldn't upgrade the user", err)
		return
	}

	rowsAffected, err := resultSQL.RowsAffected()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "ERROR  checking affected rows", err)
		return
	}

	if rowsAffected == 0 {
		respondWithError(w, 404, "ERROR  USER WAS NOT FOUND OR UNAUTHORIZED", err)
		return
	}
	
	w.WriteHeader(204)
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
		secretJWT: os.Getenv("SECRET_JWT"),
		polkaKey: os.Getenv("POLKA_KEY"),
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

	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handleGetSinglePost)

	mux.HandleFunc("GET /api/chirps", cfg.handleGetPosts)

	mux.HandleFunc("POST /api/login", cfg.handleLoginUser)

	mux.HandleFunc("POST /api/refresh", cfg.handleRefreshToken)

	mux.HandleFunc("POST /api/revoke", cfg.handleRefreshRevoke)

	mux.HandleFunc("PUT /api/users", cfg.handleUpdateUser)

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.handleDeletePost)

	mux.HandleFunc("POST /api/polka/webhooks", cfg.handleUpgradeUser)

	server.Handler = mux

	err = server.ListenAndServe()

	if err != nil {
		fmt.Println("Server error:", err)
	}
}