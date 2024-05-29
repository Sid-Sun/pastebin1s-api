package main

import (
	"encoding/base64"
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
		AllowedHeaders: []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "api_dev_key"},
		MaxAge:         30 * 60, // 30 mins of preflight caching
	}).Handler

	return handler
}

var API_DEV_KEY string

const PASTEBIN1S_HEADER_V1 string = "//<-> ) pastebin1s.com ( <-> \\\\\r\n"
const PASTEBIN1S_HEADER_V2 string = "//<-> ) pastebin1s.com V2 ( <-> \\\\\r\n"

func createRawHandler(w http.ResponseWriter, r *http.Request) {
	f := url.Values{}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		return
	}
	f.Add("api_paste_private", "1")
	f.Add("api_paste_code", encodeContent(PASTEBIN1S_HEADER_V2+string(data)))
	f.Add("api_option", "paste")
	f.Add("api_paste_expire_date", "1M")
	if r.Header.Get("api_dev_key") != "" {
		f.Add("api_dev_key", r.Header.Get("api_dev_key"))
	} else {
		f.Add("api_dev_key", API_DEV_KEY)
	}
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
	if pastebinRes.StatusCode == http.StatusOK {
		pasteID := strings.Split(string(data), "/")[3]
		w.Write([]byte(fmt.Sprintf("https://pastebin1s.com/%s\n", pasteID)))
		return
	}
	w.Write(data)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(0)
	r.ParseForm()
	if r.Form == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.Form.Get("api_dev_key") == "" {
		r.Form.Add("api_dev_key", API_DEV_KEY)
	}
	r.Form.Set("api_paste_code", encodeContent(r.Form.Get("api_paste_code")))
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
	w.Write([]byte(decodeContent(string(data))))
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
		sr.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong\n"))
		})
		sr.Get("/ping/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong\n"))
		})
	})

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	log.Printf("About to listen on %s. Go to https://0.0.0.0%s%s", listenAddr, listenAddr, basePath)
	log.Fatal(srv.ListenAndServe())
}

func encodeContent(content string) string {
	after, found := strings.CutPrefix(content, PASTEBIN1S_HEADER_V2)
	if found {
		b64 := base64.URLEncoding.EncodeToString([]byte(after))
		return PASTEBIN1S_HEADER_V2 + b64
	} // if not found, return the original content - if it fails, it fails
	// if V1, return the original content
	return content
}

func decodeContent(content string) string {
	after, found := strings.CutPrefix(content, PASTEBIN1S_HEADER_V2)
	if found {
		decoded, _ := base64.URLEncoding.DecodeString(after)
		return PASTEBIN1S_HEADER_V2 + string(decoded)
	}
	// if V1, return the content with V2 header, so frontend doesn't have to handle V1 and V2 differently (it's a mess)
	after, found = strings.CutPrefix(content, PASTEBIN1S_HEADER_V1)
	if found {
		return PASTEBIN1S_HEADER_V2 + after
	}
	return content
}
