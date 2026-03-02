package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/WagnerJust/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func splitWordAndPunctuation(word string) (prefix, core, suffix string) {
	runes := []rune(word)
	start := 0
	for start < len(runes) && !unicode.IsLetter(runes[start]){
		start++
	}
	end := len(runes)
	for end > start && !unicode.IsLetter(runes[end-1]) {
		end--
	}
	prefix = string(runes[:start])
	suffix = string(runes[end:])
	core = string(runes[start:end])

	return prefix, core, suffix
}

func cleanChirp (text string, bwords []string) string {
	if len(text) == 0 {
		return text
	}
	splitString := strings.Split(text, " ")
	for _, bw := range bwords {
		for index, w := range splitString{
			prefix, core, suffix := splitWordAndPunctuation(w)
			if strings.EqualFold(core, bw){
				core = "****"
			}
			splitString[index] = prefix + core + suffix
		}
	}
	return strings.Join(splitString, " ")
}

func respondWithError(res http.ResponseWriter, code int, msg string) error {
    return respondWithJSON(res, code, map[string]string{"error": msg})
}

func respondWithJSON(res http.ResponseWriter, code int, payload interface{}) error {
    response, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    res.Header().Set("Content-Type", "application/json")
    res.Header().Set("Access-Control-Allow-Origin", "*")
    res.WriteHeader(code)
    res.Write(response)
    return nil
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries  *database.Queries
	platform string
}

func (cfg *apiConfig) middlewareMetricsInc (next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(res, req)
	})
}

func (cfg *apiConfig) getHitCount (res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/html")
	res.WriteHeader(200)
	bodyText := fmt.Sprintf(`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`, cfg.fileserverHits.Load())
	res.Write([]byte(bodyText))
}

func (cfg *apiConfig) reset (res http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(res, 403, "Unauthorized")
		return
	}
	err := cfg.dbQueries.DeleteAllUsers(req.Context())
	if err != nil {
		respondWithError(res, 500, err.Error())
		return
	}
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	cfg.fileserverHits = atomic.Int32{}
	res.Write([]byte("Successfully reset!"))
}

func (cfg *apiConfig) createUser(res http.ResponseWriter, req *http.Request) {
	type params struct {
		Email string `json:"email"`
	}
	p := params{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&p)
	if err !=  nil {
		errString := fmt.Sprintf("Error marshalling JSON: %s", err)
		respondWithError(res, 500, errString)
		return
	}

	userParams := database.CreateUserParams{
		Email: p.Email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	user, err := cfg.dbQueries.CreateUser(req.Context(), userParams)
	if err !=  nil {
		errString := fmt.Sprintf("Error creating user: %s", err)
		respondWithError(res, 500, errString)
		return
	}
	respondWithJSON(res, 201, user)
}

func (cfg *apiConfig) createChirp (res http.ResponseWriter, req *http.Request) {
	type params struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	p := params{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&p)
	if err !=  nil {
		errString := fmt.Sprintf("Error marshalling JSON: %s", err)
		respondWithError(res, 500, errString)
		return
	}


	user, err := cfg.dbQueries.FindUserById(req.Context(), p.UserId)
	if err != nil {
		respondWithError(res, 400, err.Error())
		return
	}

	length := len(p.Body)
	if length > 140 {
		respondWithError(res, 400, "Chirp is too long")
		return
	}
	if length == 0 {
		respondWithError(res, 400, "Chirp can not be blank")
		return
	}

	bWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}
	cleanedBody := cleanChirp(p.Body, bWords)

	chirpParams := database.CreateChirpParams{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID: user.ID,
		Body: cleanedBody,
	}

	chirp, err := cfg.dbQueries.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		respondWithError(res, 500, err.Error())
		return
	}
	respondWithJSON(res, 201, chirp)
}


func addRoutes (mux *http.ServeMux, config *apiConfig) {

	// file server endpoints
	mux.Handle("GET /app/", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.Handle("GET /app/logo.png", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./assets")))))
	////////////////


	//api endpoints
	mux.Handle("GET /admin/metrics", http.HandlerFunc(config.getHitCount))
	mux.Handle("POST /admin/reset", http.HandlerFunc(config.reset))

	mux.Handle("POST /api/users", http.HandlerFunc(config.createUser))
	mux.Handle("POST /api/chirps", http.HandlerFunc(config.createChirp))
	////////////////


	mux.Handle("GET /api/healthz", http.HandlerFunc(getHealthStatus))
}

func getHealthStatus(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	res.Write([]byte("OK"))
}

func main () {
	godotenv.Load()

	// get environment
	env := os.Getenv("PLATFORM")

	// connect to db
	dbURL := os.Getenv("GOOSE_DBSTRING")
	driver := os.Getenv("GOOSE_DRIVER")
	if env == "dev" {
		log.Printf("Environment: %s", env)
		log.Printf("Database URL: %s", dbURL)
		log.Printf("Driver: %s", driver)
		fmt.Println()
		fmt.Println()
	}
	db, err := sql.Open(driver, dbURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Print("Successfully connected to database")
	defer db.Close()


	// config server
	serveMux := http.NewServeMux()
	config := &apiConfig{}
	config.platform = env
	config.dbQueries = database.New(db)

	addRoutes(serveMux, config)
	server := http.Server{
		Addr: ":8081",
		Handler: serveMux,
	}

	// start server
	log.Printf("Server at address %s", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer server.Close()
}
