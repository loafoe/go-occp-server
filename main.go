package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/loafoe/go-occp-server/api"
	"github.com/loafoe/go-occp-server/ocpp"
)

func main() {
	ocppPort := flag.String("ocpp-port", "8080", "Port for OCPP WebSocket server")
	apiPort := flag.String("api-port", "8081", "Port for HTTP API server")
	credentialsFile := flag.String("credentials-file", "", "Path to JSON file with charge point credentials")
	flag.Parse()

	server := ocpp.NewServer()

	// Load credentials if provided
	if *credentialsFile != "" {
		if err := server.Authenticator().LoadFromFile(*credentialsFile); err != nil {
			log.Fatalf("Failed to load credentials: %v", err)
		}
		log.Printf("Authentication enabled from file: %s", *credentialsFile)
	} else if err := server.Authenticator().LoadFromEnv("OCPP_CREDENTIALS"); err != nil {
		log.Fatalf("Failed to load credentials from env: %v", err)
	}

	if server.Authenticator().IsEnabled() {
		log.Println("Charge point authentication is ENABLED")
	} else {
		log.Println("WARNING: Charge point authentication is DISABLED - any charger can connect")
	}

	// OCPP WebSocket server
	ocppMux := http.NewServeMux()
	ocppMux.HandleFunc("/ocpp/", server.HandleWebSocket)

	go func() {
		addr := ":" + *ocppPort
		log.Printf("OCPP WebSocket server listening on %s", addr)
		log.Printf("Charge points should connect to: ws://YOUR_IP%s/ocpp/CHARGEPOINT_ID", addr)
		if err := http.ListenAndServe(addr, ocppMux); err != nil {
			log.Fatalf("OCPP server failed: %v", err)
		}
	}()

	// HTTP API server
	apiMux := http.NewServeMux()
	handler := api.NewHandler(server)
	handler.RegisterRoutes(apiMux)
	api.RegisterUI(apiMux)

	// Redirect root to UI
	apiMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui", http.StatusFound)
	})

	go func() {
		addr := ":" + *apiPort
		log.Printf("HTTP API server listening on %s", addr)
		if err := http.ListenAndServe(addr, apiMux); err != nil {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")
}
