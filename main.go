package main

import _ "github.com/lib/pq"
import (
	"log"
	"net/http"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"slices"
	"strings"
	"database/sql"
	"os"
	"github.com/joho/godotenv"
	"github.com/zaid-nawaz/chirpy/internal/database"
	"github.com/google/uuid"
	"time"
	"github.com/zaid-nawaz/chirpy/internal/auth"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string 	`json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func toChirpResponse(c database.Chirp) Chirp {
    return Chirp{
        ID:        c.ID,
        CreatedAt: c.CreatedAt,
        UpdatedAt: c.UpdatedAt,
        Body:      c.Body,
        UserID:    c.UserID,
    }
}


func my_func(w http.ResponseWriter, _ *http.Request){

	w.Header().Set("Content-Type" , "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))


}

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})

}

func (cfg *apiConfig) getRequestCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type" , "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())))
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))

}


func (apiCfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {

	if apiCfg.platform != "dev" {
		respondWithError(w, 403, "forbidden")
		return
	}

	err := apiCfg.db.DeleteAllUsers(r.Context())

	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	respondWithJSON(w, 200, map[string]string{
		"message": "all users deleted",
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {

	type returnVal struct {
		ErrorMessage string `json:"error"`
	}

	respBody := returnVal{
		ErrorMessage : msg,
	}

	dat, err := json.Marshal(respBody)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int , payload interface{}){

	// type returnVal struct {
	// 	CleanedBody string `json:"cleaned_body"`
	// }

	// respBody := returnVal{
	// 	CleanedBody : payload,
	// }

	dat, err := json.Marshal(payload)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)

}

func cleanedBody(body string, profane []string) string {
	wordSlice := strings.Split(body, " ")

	for i, word := range wordSlice {
		if slices.Contains(profane, strings.ToLower(word)) {
			wordSlice[i] = "****"
		}
	}

	return strings.Join(wordSlice, " ")
}

func validate_chirp(w http.ResponseWriter, r *http.Request){



}

func (apiCfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request){

	type parameters struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	params.Body = cleanedBody(params.Body, profaneWords)

	i, err := apiCfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body : params.Body,
		UserID : params.UserId,
		
	})

	payload := Chirp{
		ID : i.ID,
		CreatedAt : i.CreatedAt,
		UpdatedAt : i.UpdatedAt,
		Body : i.Body,
		UserID : i.UserID,
	}

	// type cleanedResponse struct {
	// 	CleanedBody string `json:"cleaned_body"`
	// }

	// payload := cleanedResponse{
	// 	CleanedBody : params.Body,
	// }


	respondWithJSON(w, 201, payload)
}

func (apiCfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request){

	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	HashedPassword, _ := auth.HashPassword(params.Password)

	user, err := apiCfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email : params.Email,
		HashedPassword : HashedPassword,
	})

	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	payload := User{
		ID : user.ID,
		CreatedAt : user.CreatedAt,
		UpdatedAt : user.UpdatedAt,
		Email : user.Email,
	}

	respondWithJSON(w, 201, payload)

}

func (apiCfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request){


	items , err := apiCfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	responses := make([]Chirp, len(items))

	for i , item := range items {
		responses[i] = toChirpResponse(item)
	}

	respondWithJSON(w, 200, responses)

}

func (apiCfg *apiConfig) getSingleChirpHandler(w http.ResponseWriter, r *http.Request){

	chirpIDString := r.PathValue("chirpID")

	chirpID, err := uuid.Parse(chirpIDString)


	i, err := apiCfg.db.GetChirp(r.Context(), chirpID)

	if err != nil {
		respondWithError(w, 404, err.Error())
		return
	}

	payload := Chirp{
		ID:        i.ID,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
		Body:      i.Body,
		UserID:    i.UserID,
	}

	respondWithJSON(w, 200, payload)

}

func (apiCfg *apiConfig) loginAuthentication(w http.ResponseWriter, r *http.Request){


	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	i, err := apiCfg.db.GetUserByEmail(r.Context(), params.Email)

	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	matched, err := auth.CheckPasswordHash(params.Password, i.HashedPassword)

	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	if !matched {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}



	
	payload := User{
		ID : i.ID,
		CreatedAt : i.CreatedAt,
		UpdatedAt : i.UpdatedAt,
		Email : i.Email,
	}

	respondWithJSON(w, 200, payload)
	

}


func main() {

	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	_ = dbQueries

	mux := http.NewServeMux()



	server := &http.Server{
		Addr : ":8080",
		Handler : mux,
	}

	
	apiCfg := &apiConfig{
		db : dbQueries,
		platform : platform,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	

	mux.HandleFunc("GET /api/healthz", my_func)
	mux.HandleFunc("GET /admin/metrics", apiCfg.getRequestCount)

	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	mux.HandleFunc("POST /api/validate_chirp", validate_chirp)

	mux.HandleFunc("POST /api/users", apiCfg.createUserHandler )

	mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler )

	mux.HandleFunc("GET /api/chirps", apiCfg.getChirpHandler )

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getSingleChirpHandler )

	mux.HandleFunc("POST /api/login", apiCfg.loginAuthentication )
	

	log.Fatal(server.ListenAndServe())
	
}

//connection string - postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable
