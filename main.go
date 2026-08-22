package main

import (
	"fmt"
	"net/http"
)

func main() {
	staticDir := "./static"

	fileServer := http.FileServer(http.Dir(staticDir))

	http.Handle("/", fileServer)

	port := ":8080"
	fmt.Printf("Servidor estatico escuchano en http://localhost%s\n", port)
	fmt.Printf("Sirviendo archios desde: %s\n", staticDir)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}
