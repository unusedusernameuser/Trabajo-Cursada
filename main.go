package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	staticDir := "./static"
	fileServer := http.FileServer(http.Dir(staticDir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))

		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			serve404(w, staticDir)
			return
		}
		if err != nil {
			log.Printf("error al acceder a %s: %v", path, err)
			http.Error(w, "error interno del servidor", http.StatusInternalServerError)
			return
		}

		if info.IsDir() && r.URL.Path != "/" {
			indexPath := filepath.Join(path, "index.html")
			if _, errIndex := os.Stat(indexPath); os.IsNotExist(errIndex) {
				serve404(w, staticDir)
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})

	port := ":8080"
	log.Printf("Servidor estatico escuchando en http://localhost%s\n", port)
	log.Printf("Sirviendo archivos desde: %s\n", staticDir)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error al iniciar el servidor: %s\n", err)
	}
}

func serve404(w http.ResponseWriter, staticDir string) {
	content, err := os.ReadFile(filepath.Join(staticDir, "404.html"))
	if err != nil {
		http.Error(w, "404 page not fond", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write(content)
}
