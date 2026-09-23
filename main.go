package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/GitShinobi/Chirpy/internal/auth"
	"github.com/GitShinobi/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	Db             *database.Queries
	Platform       string
	Secret 		   string
	Polka 		   string
	FileserverHits atomic.Int32
}
type errorVals struct {
	Error string `json:"error"`
}
type createChirpParams struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}
type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}
type userParams struct {
	Password 			string `json:"password"`
	Email    			string `json:"email"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token 	  string 	`json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed bool `json:"is_chirpy_red"`  
}
type Bearer struct{
	Token string `json:"token"`
}

type webhooksParams struct {
	Event   string    `json:"event"`
	Data struct{
		UserID  uuid.UUID `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) handlerMetrics(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.FileserverHits.Load())
	_, err := writer.Write([]byte(html))
	if err != nil {
		fmt.Println(err.Error())
	}
}
func (cfg *apiConfig) handlerReset(writer http.ResponseWriter, req *http.Request) {
	if cfg.Platform != "dev" {
		writer.WriteHeader(403)
		return
	}
	err := cfg.Db.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(writer, 400, "Error deleting users")
		return
	}
	cfg.FileserverHits.Store(0)
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
}
func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	dbURL := os.Getenv("DB_URL")
	dbPlatform := os.Getenv("PLATFORM")
	dbSecret := os.Getenv("SECRET_KEY")
	dbPolka := os.Getenv("POLKA_KEY")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{
		Db:       dbQueries,
		Platform: dbPlatform,
		Secret: dbSecret,
		Polka: dbPolka,
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLoginUser)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke ", apiCfg.handlerRevoke)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirp)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpById)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirp)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerWebhooks)
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err.Error())
	}

}
func handlerReadiness(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	_, err := writer.Write([]byte("OK"))
	if err != nil {
		fmt.Println(err.Error())
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	errStr := errorVals{msg}
	errStrData, err := json.Marshal(errStr)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
		return
	}
	w.WriteHeader(code)
	_, err = w.Write(errStrData)
	if err != nil {
		w.Write([]byte(`{"error": "Error writing in the body"}`))
		return
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if payload == nil{
		w.WriteHeader(code)
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
		}
	w.WriteHeader(code)
	if string(data) != "null"{
		_, err = w.Write(data)
		if err != nil {
			respondWithError(w, 400, fmt.Sprintf("Error writing %s in the body: %s\n", data, err))
			return
		}	
	}
}

func replaceProfane(text string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	textTab := strings.Split(text, " ")
	for _, badWord := range badWords {
		for i, word := range textTab {
			if badWord == strings.ToLower(word) {
				textTab[i] = "****"
			}
		}
	}
	return strings.Join(textTab, " ")
}

func (cfg *apiConfig) handlerCreateUser(writer http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	params := userParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("Error hashing password: %s", err.Error()))
		return
	}
	dbUser, err := cfg.Db.CreateUser(req.Context(), database.CreateUserParams{HashedPassword: hashedPassword, Email: params.Email})
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("Error creating user: %s", err.Error()))
		return
	}
	user := User{
		ID:             dbUser.ID,
		CreatedAt:      dbUser.CreatedAt,
		UpdatedAt:      dbUser.UpdatedAt,
		Email:          dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
	respondWithJSON(writer, 201, user)
}

func (cfg *apiConfig) handlerCreateChirp(writer http.ResponseWriter, req *http.Request) {
	UserToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("Error : %s", err.Error()))
		return
	}
	userId, err := auth.ValidateJWT(UserToken,cfg.Secret)
	if err != nil {
		respondWithError(writer, 401, fmt.Sprintf("Unauthorized : %s", err.Error()))
		return
	}
	decoder := json.NewDecoder(req.Body)
	chirpParam := createChirpParams{
		UserID: userId,
	}
	err = decoder.Decode(&chirpParam)
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("Error decoding parameters: %s\n", err))
		return
	}
	if len(chirpParam.Body) > 140 {
		respondWithError(writer, 400, "Chirp is too long\n")
		return
	}
	chirpParam.Body = replaceProfane(chirpParam.Body)
	dbChirp, err := cfg.Db.CreateChirp(req.Context(), database.CreateChirpParams{Body: chirpParam.Body, UserID: chirpParam.UserID})
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("Error creating chirp: %s", err.Error()))
		return
	}
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	respondWithJSON(writer, 201, chirp)
}

func (cfg *apiConfig) handlerGetChirp(writer http.ResponseWriter, req *http.Request) {
	authorId := req.URL.Query().Get("author_id")
	sortParam := req.URL.Query().Get("sort")
	chirps := []Chirp{}
	var dbChirps []database.Chirp
	if len(authorId)>0{
		parsedAuthorID, err := uuid.Parse(authorId)
		if err != nil {
			respondWithError(writer, 400, err.Error())
			return
		}
		tempChirps, err := cfg.Db.GetChirpsByUserId(req.Context(),parsedAuthorID)
		if err != nil {
			respondWithError(writer, 404, fmt.Sprintf("Error getting chirps: %s", err.Error()))
			return
		}
		dbChirps = tempChirps
	}else{
		tempChirps, err  := cfg.Db.GetChirps(req.Context())
		if err != nil {
			respondWithError(writer, 404, fmt.Sprintf("Error getting chirps: %s", err.Error()))
			return
		}
		dbChirps = tempChirps
	}
	if len(sortParam) > 0 && sortParam == "desc"{
		sort.Slice(dbChirps, func(i, j int) bool {
			return dbChirps[i].CreatedAt.After(dbChirps[j].CreatedAt)
		})
	}
	for _, dbChirp := range dbChirps {
		chirp := Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		}
		chirps = append(chirps, chirp)
	}
	respondWithJSON(writer, 200, chirps)

}

func (cfg *apiConfig) handlerGetChirpById(writer http.ResponseWriter, req *http.Request) {
	id := req.PathValue("chirpID")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("Invalid ID %s", err.Error()))
		return
	}
	dbChirp, err := cfg.Db.GetChirpsById(req.Context(), parsedID)
	if err != nil {
		respondWithError(writer, 404, fmt.Sprintf("Error getting chirps %s", err.Error()))
		return
	}
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	respondWithJSON(writer, 200, chirp)
}

func (cfg *apiConfig) handlerLoginUser(writer http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	params := userParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	dbUser,err := cfg.Db.GetUserByEmail(req.Context(),params.Email)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	match, err := auth.CheckPasswordHash(params.Password,dbUser.HashedPassword)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	if !match {
		respondWithError(writer, 401, "Incorrect email or password")
		return
	}
	expireToken := time.Second*60*60
	UserToken,err := auth.MakeJWT(dbUser.ID,cfg.Secret,expireToken)
	if  err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	expireRefreshToken := time.Second*60*60*24*60
	refreshToken := auth.MakeRefreshToken()
	cfg.Db.CreateRefreshToken(context.Background(),database.CreateRefreshTokenParams{
		Token   :refreshToken,
		UserID  : dbUser.ID,
		ExpiresAt: time.Now().Add(expireRefreshToken),
	})
	user := User{
		ID:             dbUser.ID,
		CreatedAt:      dbUser.CreatedAt,
		UpdatedAt:      dbUser.UpdatedAt,
		Email:          dbUser.Email,
		Token: 			UserToken,
		RefreshToken:   refreshToken,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
	respondWithJSON(writer, 200, user)
}
func (cfg *apiConfig) handlerRefresh(writer http.ResponseWriter, req *http.Request) {
	tokenStr,err := auth.GetBearerToken(req.Header) 
	if err != nil{
		respondWithError(writer, 401, err.Error())
		return
	}
	refreshToken,err := cfg.Db.GetUserFromRefreshToken(context.Background(),tokenStr)
	if err != nil {
		respondWithError(writer, 401, err.Error())
		return
	}
	if refreshToken.ExpiresAt.Before(time.Now()) || refreshToken.RevokedAt.Valid {
		respondWithError(writer, 401, "token expirée")
		return
	}
	bearerToken := Bearer{Token: tokenStr}
	writer.WriteHeader(200)
	respondWithJSON(writer,200, bearerToken)
}


func (cfg *apiConfig) handlerRevoke(writer http.ResponseWriter, req *http.Request) {
	UserToken,err := auth.GetBearerToken(req.Header) 
	if err != nil{
		respondWithError(writer, 401, err.Error())
		return
	}
	userId, err := auth.ValidateJWT(UserToken,cfg.Secret)
	if err != nil {
		respondWithError(writer, 401, fmt.Sprintf("Unauthorized : %s", err.Error()))
		return
	}
	err = cfg.Db.AddrevokedAt(context.Background(),userId)
	if err != nil {
		respondWithError(writer, 500, err.Error())
		return
	}
		writer.WriteHeader(204)


}

func (cfg *apiConfig) handlerUpdateUser(writer http.ResponseWriter, req *http.Request) {
	UserToken,err := auth.GetBearerToken(req.Header) 
	if err != nil{
		respondWithError(writer, 401, err.Error())
		return
	}
	userId, err := auth.ValidateJWT(UserToken,cfg.Secret)
	if err != nil {
		respondWithError(writer, 401, fmt.Sprintf("Unauthorized : %s", err.Error()))
		return
	}
	decoder := json.NewDecoder(req.Body)
	params := userParams{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	if params.Email == "" || params.Password == ""{
		respondWithError(writer, 400, "error email and password required")
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("Error hashing password: %s", err.Error()))
		return
	}
	dbUser, err := cfg.Db.UpdateUserEmailAndPassword(req.Context(), database.UpdateUserEmailAndPasswordParams{
		HashedPassword: hashedPassword, 
		Email: params.Email,
		ID:userId,})
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("Error updating user: %s", err.Error()))
		return
	}
	user := User{
		ID:             dbUser.ID,
		CreatedAt:      dbUser.CreatedAt,
		UpdatedAt:      dbUser.UpdatedAt,
		Email:          dbUser.Email,
		Token: 			UserToken,
		RefreshToken: 	UserToken,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
	respondWithJSON(writer, 200, user)
}

func (cfg *apiConfig) handlerDeleteChirp(writer http.ResponseWriter, req *http.Request) {
	id := req.PathValue("chirpID")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("Invalid ID %s", err.Error()))
		return
	}
	dbChirp, err := cfg.Db.GetChirpsById(req.Context(), parsedID)
	if err != nil {
		respondWithError(writer, 404, fmt.Sprintf("Error getting chirps %s", err.Error()))
		return
	}
	UserToken,err := auth.GetBearerToken(req.Header) 
	if err != nil{
		respondWithError(writer, 401, err.Error())
		return
	}
	
	userId, err := auth.ValidateJWT(UserToken,cfg.Secret)
	if err != nil {
		respondWithError(writer, 401, fmt.Sprintf("Unauthorized : %s", err.Error()))
		return
	}
	if userId != dbChirp.UserID{
		respondWithError(writer, 403, "You can only delete your own chirps")
		return
	}
	err = cfg.Db.DeleteChirpsById(req.Context(), parsedID)
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("Error deleting chirps %s", err.Error()))
		return
	}

		writer.WriteHeader(204)
}

func (cfg *apiConfig) handlerWebhooks(writer http.ResponseWriter, req *http.Request) {
	polkaKey,err := auth.GetAPIKey(req.Header) 
	if err != nil{
		respondWithError(writer, 401, err.Error())
		return
	}
	if polkaKey != cfg.Polka{
		respondWithError(writer, 401, "incorect API key")
		return
	}
	decoder := json.NewDecoder(req.Body)
	params := webhooksParams{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	if params.Event != "user.upgraded"{
		writer.WriteHeader(204)
		return
	}
	userId := params.Data.UserID
	err = cfg.Db.UpdateChirpyRedMembership(context.Background(),userId)
	if err != nil {
		respondWithError(writer, 404, fmt.Sprintf("Error updating membership %s", err.Error()))
		return
	}
	writer.WriteHeader(204)

}

