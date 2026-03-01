package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"unicode"
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

func (cfg *apiConfig) resetHitCount (res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	cfg.fileserverHits = atomic.Int32{}
	res.Write([]byte("Successfully reset hit count!"))
}


func validateChirp (res http.ResponseWriter, req *http.Request) {
	type params struct {
		Body string `json:"body"`
	}
	p := params{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&p)
	if err != nil {
		errString := fmt.Sprintf("Error marshalling JSON: %s", err)
		respondWithError(res, 500, errString)
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

	payload := struct {
		valid bool
		CleanedBody string `json:"cleaned_body"`
	} {
		valid: true,
		CleanedBody: cleanedBody,
	}
	respondWithJSON(res, 200, payload)

}

func addRoutes (mux *http.ServeMux, config *apiConfig) {

	// file server endpoints
	mux.Handle("GET /app/", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.Handle("GET /app/logo.png", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./assets")))))
	////////////////


	//api endpoints
	mux.Handle("GET /admin/metrics", http.HandlerFunc(config.getHitCount))
	mux.Handle("POST /admin/reset", http.HandlerFunc(config.resetHitCount))

	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validateChirp))
	////////////////


	mux.Handle("GET /api/healthz", http.HandlerFunc(getHealthStatus))
}

func getHealthStatus(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	res.Write([]byte("OK"))
}

func main () {
	serveMux := http.NewServeMux()
	config := &apiConfig{}
	addRoutes(serveMux, config)
	server := http.Server{
		Addr: ":8081",
		Handler: serveMux,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer server.Close()


}
