package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

func processIconURL(icon string) string {
	if len(icon) > 3 && icon[:3] == "sh:" {
		appName := icon[3:]
		ext := "svg" // Default extension

		if lastDotIndex := strings.LastIndex(appName, "."); lastDotIndex != -1 && lastDotIndex < len(appName)-1 {
			potentialExt := appName[lastDotIndex+1:]
			switch potentialExt {
			case "png", "svg", "webp":
				ext = potentialExt
				appName = appName[:lastDotIndex]
			}
		}
		
		return fmt.Sprintf("https://cdn.jsdelivr.net/gh/selfhst/icons@master/%s/%s.%s", ext, appName, ext)
	}
	return icon
}

func (sp *SleepProxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if sp.isStoppingState() {
		w.Write([]byte(`{"status":"stopping"}`))
		return
	}

	if !sp.areContainersUp() {
		w.Write([]byte(`{"status":"sleeping"}`))
		return
	}

	if sp.checkContainersReady(ctx) {
		w.Write([]byte(`{"status":"ready"}`))
	} else {
		// Update activity to prevent timeout during startup
		sp.updateActivity()
		w.Write([]byte(`{"status":"starting"}`))
	}
}

func (sp *SleepProxy) serveLoadingPage(w http.ResponseWriter, r *http.Request) {
	// Read the HTML template from static files
	htmlContent, err := staticFiles.ReadFile("static/loading.html")
	if err != nil {
		log.Printf("Failed to read loading.html: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Inject check interval and endpoint prefix as meta tags
	checkIntervalMs := int(sp.config.CheckInterval.Milliseconds())

	indicatorHTML := `<div class="spinner"></div>`
	if sp.config.TargetServiceIcon != "" {
		iconURL := processIconURL(sp.config.TargetServiceIcon)
		indicatorHTML = fmt.Sprintf(`<img src="%s" class="service-icon" alt="Service Icon">`, iconURL)
	}

	html := fmt.Sprintf(string(htmlContent), checkIntervalMs, sp.config.EndpointPrefix, indicatorHTML, sp.config.TargetServiceDisplayName)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (sp *SleepProxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	// Update activity timestamp
	sp.updateActivity()

	log.Printf("Proxying request: %s %s", r.Method, r.URL.Path)

	// Check if containers are up
	ctx := context.Background()
	if !sp.areContainersUp() {
		log.Printf("Containers are down, starting them...")
		if err := sp.startContainers(ctx); err != nil {
			log.Printf("Failed to start containers: %v", err)
			http.Error(w, "Failed to start services", http.StatusInternalServerError)
			return
		}
		sp.setContainersUp(true)
	}

	// Check if containers are ready
	if !sp.checkContainersReady(ctx) {
		log.Printf("Containers not ready yet, showing loading page")
		sp.serveLoadingPage(w, r)
		return
	}

	// Create the reverse proxy
	targetURL := fmt.Sprintf("http://%s:%s", sp.config.TargetService, sp.config.TargetPort)
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Printf("Failed to parse target URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		sp.setContainersUp(false)
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
	}

	proxy.ServeHTTP(w, r)
}

func (sp *SleepProxy) handleShutdown(w http.ResponseWriter, r *http.Request) {
	log.Printf("Manual shutdown requested")

	ctx := context.Background()
	if err := sp.stopContainers(ctx); err != nil {
		log.Printf("Failed to stop containers: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error":"Failed to stop containers: %v"}`, err)))
		return
	}

	sp.setContainersUp(false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"Containers stopped"}`))
}

func (sp *SleepProxy) setupRoutes() {
	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	// API endpoints with prefix
	prefix := "/" + sp.config.EndpointPrefix
	http.HandleFunc(prefix+"/health", sp.handleHealth)
	http.HandleFunc(prefix+"/shutdown", sp.handleShutdown)

	// Main proxy handler
	http.HandleFunc("/", sp.handleProxy)
}
