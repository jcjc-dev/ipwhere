// Package main IPWhere API
//
// An all-in-one IP geolocation lookup server.
//
//	@title			IPWhere API
//	@version		1.0
//	@description	IP geolocation lookup service using DB-IP database
//
//	@contact.name	IPWhere
//	@contact.url	https://github.com/Shoyu-Dev/ipwhere
//
//	@license.name	MIT (Code) / CC BY 4.0 (Data)
//	@license.url	https://github.com/Shoyu-Dev/ipwhere/blob/main/LICENSE
//
//	@host			localhost:8080
//	@BasePath		/
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Shoyu-Dev/ipwhere/internal/api"
	"github.com/Shoyu-Dev/ipwhere/internal/geo"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

//go:embed static/*
var staticFiles embed.FS

const (
	defaultListenAddr = ":8080"
)

func main() {
	// Parse command line flags
	listenAddr := flag.String("l", "", "Address to listen on (default :8080)")
	flag.StringVar(listenAddr, "listen", "", "Address to listen on (default :8080)")

	headless := flag.Bool("H", false, "Run in headless mode (API only, no frontend)")
	flag.BoolVar(headless, "headless", false, "Run in headless mode (API only, no frontend)")

	enableOnlineFeatures := flag.Bool("online", false, "Enable online features (reverse DNS lookup)")
	flag.BoolVar(enableOnlineFeatures, "enable-online-features", false, "Enable online features (reverse DNS lookup)")

	cityDBPath := flag.String("city-db", "", "Path to city MMDB database")
	asnDBPath := flag.String("asn-db", "", "Path to ASN MMDB database")

	flag.Parse()

	// Check environment variables
	if *listenAddr == "" {
		*listenAddr = os.Getenv("LISTEN_ADDR")
	}
	if *listenAddr == "" {
		*listenAddr = defaultListenAddr
	}

	if !*headless {
		headlessEnv := os.Getenv("HEADLESS")
		*headless = headlessEnv == "true" || headlessEnv == "1"
	}

	if !*enableOnlineFeatures {
		onlineEnv := os.Getenv("ENABLE_ONLINE_FEATURES")
		*enableOnlineFeatures = onlineEnv == "true" || onlineEnv == "1"
	}

	// Determine database paths
	if *cityDBPath == "" {
		*cityDBPath = os.Getenv("CITY_DB_PATH")
	}
	if *cityDBPath == "" {
		// Default: look in same directory as executable, then /app/data
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		candidates := []string{
			filepath.Join(execDir, "data", "dbip-city-lite.mmdb"),
			"/app/data/dbip-city-lite.mmdb",
			"data/dbip-city-lite.mmdb",
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				*cityDBPath = p
				break
			}
		}
	}

	if *asnDBPath == "" {
		*asnDBPath = os.Getenv("ASN_DB_PATH")
	}
	if *asnDBPath == "" {
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		candidates := []string{
			filepath.Join(execDir, "data", "dbip-asn-lite.mmdb"),
			"/app/data/dbip-asn-lite.mmdb",
			"data/dbip-asn-lite.mmdb",
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				*asnDBPath = p
				break
			}
		}
	}

	if *cityDBPath == "" || *asnDBPath == "" {
		log.Fatal("Database files not found. Please provide paths via --city-db and --asn-db flags or CITY_DB_PATH and ASN_DB_PATH environment variables")
	}

	// Check if running in CLI mode (IP argument provided)
	args := flag.Args()
	cliMode := len(args) > 0

	if !cliMode {
		log.Printf("Using city database: %s", *cityDBPath)
		log.Printf("Using ASN database: %s", *asnDBPath)
	}

	// Initialize geo reader
	geoReader, err := geo.NewReader(*cityDBPath, *asnDBPath, *enableOnlineFeatures)
	if err != nil {
		if cliMode {
			fmt.Fprintf(os.Stderr, "Error: failed to initialize geo reader: %v\n", err)
			os.Exit(1)
		}
		log.Fatalf("Failed to initialize geo reader: %v", err)
	}
	defer geoReader.Close()

	// CLI mode: lookup the IP and print result
	if cliMode {
		runCLI(geoReader, args[0])
		return
	}

	// Create router
	r := api.NewRouter()

	// Setup API routes
	handler := api.NewHandler(geoReader, *enableOnlineFeatures)
	handler.SetupRoutes(r)

	// Setup Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Serve frontend if not headless
	if !*headless {
		log.Println("Frontend enabled")
		setupFrontend(r)
	} else {
		log.Println("Running in headless mode (API only)")
	}

	// Start server
	log.Printf("Starting server on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func setupFrontend(r *chi.Mux) {
	// Get the static subdirectory from embedded files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to get static files: %v", err)
	}

	// Serve static files
	fileServer := http.FileServer(http.FS(staticFS))

	// Serve index.html for root and unmatched routes (SPA support)
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// Try to open the file
		path := req.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		f, err := staticFS.Open(path[1:]) // Remove leading slash
		if err != nil {
			// File not found, serve index.html for SPA
			req.URL.Path = "/index.html"
		} else {
			f.Close()
		}

		fileServer.ServeHTTP(w, req)
	})
}

// runCLI performs a direct IP lookup and prints the result as JSON
func runCLI(geoReader *geo.Reader, ipStr string) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		fmt.Fprintf(os.Stderr, "Error: invalid IP address: %s\n", ipStr)
		os.Exit(1)
	}

	info, err := geoReader.Lookup(ip)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: lookup failed: %v\n", err)
		os.Exit(1)
	}

	output, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to format output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}
