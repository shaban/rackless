package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	dev := flag.Bool("dev", false, "Enable development mode")
	port := flag.String("port", "8080", "Port to serve on")
	flag.Parse()

	if *dev {
		log.Println("Starting Rackless development server...")
		log.Printf("Server will be available at http://localhost:%s", *port)
	}

	// TODO: Implement server
	// - Serve WASM app
	// - Provide AudioUnit API endpoints
	// - Handle parameter mapping requests

	log.Printf("Server starting on port %s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
