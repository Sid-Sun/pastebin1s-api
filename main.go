package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

func WithCors() func(h http.Handler) http.Handler {
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "X-Requested-With", "Content-Type", "Accept"},
		MaxAge:         30 * 60, // 30 mins of preflight caching
	}).Handler

	return handler
}

var API_DEV_KEY string

func createRawHandler(w http.ResponseWriter, r *http.Request) {
	f := url.Values{}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	f.Add("api_paste_private", "1")
	f.Add("api_paste_code", string(data))
	f.Add("api_option", "paste")
	f.Add("api_paste_expire_date", "1M")
	r.Form.Add("api_dev_key", API_DEV_KEY)
	d1 := f.Encode()
	pastebinReq, err := http.NewRequest(http.MethodPost, "https://pastebin.com/api/api_post.php", strings.NewReader(d1))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	pastebinReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	pastebinRes, err := http.DefaultClient.Do(pastebinReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	data, err = io.ReadAll(pastebinRes.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	w.WriteHeader(pastebinRes.StatusCode)
	w.Write(data)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(0)
	r.ParseForm()
	if r.Form == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// r.Form.Add("api_user_key", "2084f8a46a3246b4fe4023eaa050723b")
	r.Form.Add("api_dev_key", API_DEV_KEY)
	d1 := r.Form.Encode()
	pastebinReq, err := http.NewRequest(http.MethodPost, "https://pastebin.com/api/api_post.php", strings.NewReader(d1))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	pastebinReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	pastebinRes, err := http.DefaultClient.Do(pastebinReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	data, err := io.ReadAll(pastebinRes.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	w.WriteHeader(pastebinRes.StatusCode)
	w.Write(data)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	pasteKey := chi.URLParam(r, "paste_key")
	pastebinReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://pastebin.com/raw/%s", pasteKey), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	pastebinRes, err := http.DefaultClient.Do(pastebinReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	data, err := io.ReadAll(pastebinRes.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	w.WriteHeader(pastebinRes.StatusCode)
	w.Write(data)
}

func main() {
	API_DEV_KEY = os.Getenv("API_DEV_KEY")
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("APP_PORT"); ok {
		listenAddr = ":" + val
	}
	r := chi.NewRouter()
	r.Use(WithCors())

	basePath := os.Getenv("BASE_PATH")
	if basePath == "" {
		basePath = "/"
	}
	r.Route(basePath, func(sr chi.Router) {
		sr.Post("/create", createHandler)
		sr.Put("/raw/{file}", createRawHandler)
		sr.Put("/raw", createRawHandler)
		sr.Get("/get/{paste_key}", getHandler)
	})

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	log.Printf("About to listen on %s. Go to https://0.0.0.0%s%s", listenAddr, listenAddr, basePath)
	log.Fatal(srv.ListenAndServe())
}
