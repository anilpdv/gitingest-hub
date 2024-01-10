package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"gitingest-hub/pkg/server"
)

func main() {
	port := flag.String("port", "8080", "Port to run the HTTP server on")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	srv, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	mux := http.NewServeMux()

	// Static Assets
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Application Routes
	mux.HandleFunc("/", srv.HandleIndex)
	mux.HandleFunc("/api/ingest", srv.HandleIngest)
	mux.HandleFunc("/api/rechunk", srv.HandleRechunk)
	mux.HandleFunc("/api/download", srv.HandleDownload)

	addr := ":" + *port
	fmt.Printf("\n⚡ GitIngest Hub running at: http://localhost:%s\n", *port)
	fmt.Printf("🎨 Neubrutalism UI + In-Memory Tarball Streamer + Chrome Built-in AI Active\n\n")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
