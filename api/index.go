package rssfunction

import (
	"encoding/json"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/beevik/etree"
)

const PODCAST_URL = "https://anchor.fm/s/49f0c604/podcast/rss"

type PodcastItem struct {
	Title        string `json:"name"`
	EnclosureURL string `json:"url"`
	ImageURL     string `json:"cover_art_url"`
}

func fetchPodcastData() (string, error) {
	resp, err := http.Get(PODCAST_URL)
	if err != nil {
		return "", fmt.Errorf("erro ao buscar o feed RSS: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler o corpo da resposta: %v", err)
	}

	return string(body), nil
}

func parsePodcastData(xmlText string) ([]PodcastItem, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlText); err != nil {
		return nil, fmt.Errorf("erro ao analisar o XML: %v", err)
	}

	var podcastData []PodcastItem

	for _, item := range doc.FindElements("//item") {
		title := item.SelectElement("title").Text()
		enclosureURL := item.SelectElement("enclosure").Attr[0].Value
		imageURL := "Imagem indisponível"
		if img := item.SelectElement("itunes:image"); img != nil {
			imageURL = img.Attr[0].Value
		}

		podcastData = append(podcastData, PodcastItem{
			Title:        title,
			EnclosureURL: enclosureURL,
			ImageURL:     imageURL,
		})
	}

	return podcastData, nil
}

// Middleware para adicionar compressão Gzip
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func GetPodcast(w http.ResponseWriter, r *http.Request) {
    setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	xmlText, err := fetchPodcastData()
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao buscar o feed do podcast: %v", err), http.StatusInternalServerError)
		return
	}

	podcastData, err := parsePodcastData(xmlText)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao analisar os dados do podcast: %v", err), http.StatusInternalServerError)
		return
	}

	// Minificar JSON sem espaços extras
	w.Header().Set("Content-Type", "application/json")
	jsonBytes, err := json.Marshal(podcastData)
	if err != nil {
		http.Error(w, "Erro ao gerar JSON", http.StatusInternalServerError)
		return
	}

	w.Write(jsonBytes)
}

