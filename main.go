package main

import (
	"log"
	"net/http"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"slices"
	"strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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


func (cfg *apiConfig) setCountZero(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type" , "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
	w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())))

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

	type parameters struct {
		Body string `json:"body"`
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

	type cleanedResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	payload := cleanedResponse{
		CleanedBody : params.Body,
	}


	respondWithJSON(w, 200, payload)

}

func main() {
	mux := http.NewServeMux()



	server := &http.Server{
		Addr : ":8080",
		Handler : mux,
	}

	
	apiCfg := &apiConfig{}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	

	mux.HandleFunc("GET /api/healthz", my_func)
	mux.HandleFunc("GET /admin/metrics", apiCfg.getRequestCount)

	mux.HandleFunc("POST /admin/reset", apiCfg.setCountZero)

	mux.HandleFunc("POST /api/validate_chirp", validate_chirp)

	

	log.Fatal(server.ListenAndServe())
	
}
