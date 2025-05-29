package webui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"strings"
	"time"

	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
)

//go:embed static/*
var staticFiles embed.FS

// WebUIOptions contains configuration for WebUI
type WebUIOptions struct {
	// View to use for rendering
	View *WebView

	// Tileset configuration
	TilesetPath string
	Tileset     *TilesetConfig

	// Server configuration
	ListenAddr  string
	PollTimeout time.Duration

	// CORS settings
	AllowOrigins []string

	// Static file serving
	StaticPath string // Optional: override embedded files
}

// WebUI provides a web-based interface for dgclient
type WebUI struct {
	view       *WebView
	tileset    *TilesetConfig
	rpcHandler *RPCHandler
	mux        *http.ServeMux
	options    WebUIOptions
}

// NewWebUI creates a new WebUI instance
func NewWebUI(opts WebUIOptions) (*WebUI, error) {
	webui := &WebUI{
		view:    opts.View,
		options: opts,
		mux:     http.NewServeMux(),
	}

	// Load tileset if specified
	if opts.Tileset != nil {
		webui.tileset = opts.Tileset
	} else if opts.TilesetPath != "" {
		tileset, err := LoadTilesetConfig(opts.TilesetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load tileset: %w", err)
		}
		webui.tileset = tileset
	}

	// Set tileset on view if available
	if webui.view != nil && webui.tileset != nil {
		webui.view.SetTileset(webui.tileset)
	}

	// Create RPC handler
	webui.rpcHandler = NewRPCHandler(webui)

	// Set up routes
	webui.setupRoutes()

	return webui, nil
}

// setupRoutes configures HTTP routes
func (w *WebUI) setupRoutes() {
	// RPC endpoint
	w.mux.HandleFunc("/rpc", w.handleRPC)

	// Tileset image endpoint
	w.mux.HandleFunc("/tileset/image", w.handleTilesetImage)

	// Static files
	if w.options.StaticPath != "" {
		// Serve from filesystem
		w.mux.Handle("/", http.FileServer(http.Dir(w.options.StaticPath)))
	} else {
		// Serve embedded files
		w.mux.Handle("/", http.FileServer(http.FS(staticFiles)))
	}
}

// ServeHTTP implements http.Handler
func (w *WebUI) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.addCORSHeaders(rw, r)

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	// Route request
	w.mux.ServeHTTP(rw, r)
}

// addCORSHeaders adds CORS headers to response
func (w *WebUI) addCORSHeaders(rw http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// Check if origin is allowed
	if w.isOriginAllowed(origin) {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
	} else if len(w.options.AllowOrigins) == 0 {
		// If no origins specified, allow all
		rw.Header().Set("Access-Control-Allow-Origin", "*")
	}

	rw.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	rw.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	rw.Header().Set("Access-Control-Max-Age", "86400")

	// Prevent caching of dynamic content
	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	rw.Header().Set("Pragma", "no-cache")
	rw.Header().Set("Expires", "0")
}

// isOriginAllowed checks if an origin is in the allowed list
func (w *WebUI) isOriginAllowed(origin string) bool {
	for _, allowed := range w.options.AllowOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

// handleRPC processes JSON-RPC requests
func (w *WebUI) handleRPC(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON-RPC request
	var rpcReq RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		w.sendRPCError(rw, nil, ParseError, "Parse error")
		return
	}

	// Validate JSON-RPC version
	if rpcReq.JSONRPC != "2.0" {
		w.sendRPCError(rw, rpcReq.ID, InvalidRequest, "Invalid Request")
		return
	}

	// Process request
	ctx := r.Context()
	response := w.rpcHandler.HandleRequest(ctx, &rpcReq)

	// Send response
	rw.Header().Set("Content-Type", "application/json")

	// Set compression header
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		rw.Header().Set("Content-Encoding", "gzip")
	}

	json.NewEncoder(rw).Encode(response)
}

// sendRPCError sends a JSON-RPC error response
func (w *WebUI) sendRPCError(rw http.ResponseWriter, id interface{}, code int, message string) {
	response := &RPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK) // JSON-RPC errors still return 200
	json.NewEncoder(rw).Encode(response)
}

// handleTilesetImage serves the tileset image
func (w *WebUI) handleTilesetImage(rw http.ResponseWriter, r *http.Request) {
	if w.tileset == nil || w.tileset.GetImageData() == nil {
		http.NotFound(rw, r)
		return
	}

	// Check for If-None-Match header for caching
	etag := fmt.Sprintf(`"%s-%s"`, w.tileset.Name, w.tileset.Version)
	if r.Header.Get("If-None-Match") == etag {
		rw.WriteHeader(http.StatusNotModified)
		return
	}

	// Set caching headers
	rw.Header().Set("ETag", etag)
	rw.Header().Set("Cache-Control", "public, max-age=3600")
	rw.Header().Set("Content-Type", "image/png")

	// Encode image as PNG
	if err := png.Encode(rw, w.tileset.GetImageData()); err != nil {
		http.Error(rw, "Failed to encode image", http.StatusInternalServerError)
		return
	}
}

// GetTileset returns the current tileset configuration
func (w *WebUI) GetTileset() *TilesetConfig {
	return w.tileset
}

// UpdateTileset updates the tileset configuration
func (w *WebUI) UpdateTileset(tileset *TilesetConfig) error {
	w.tileset = tileset

	if w.view != nil {
		w.view.SetTileset(tileset)
	}

	return nil
}

// SetView sets the view for the WebUI
func (w *WebUI) SetView(view *WebView) {
	w.view = view

	if w.tileset != nil {
		view.SetTileset(w.tileset)
	}
}

// GetView returns the current view
func (w *WebUI) GetView() *WebView {
	return w.view
}

// Start starts the WebUI server
func (w *WebUI) Start(addr string) error {
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      w,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("WebUI server starting on %s\n", addr)
	return server.ListenAndServe()
}

// StartWithContext starts the WebUI server with context for graceful shutdown
func (w *WebUI) StartWithContext(ctx context.Context, addr string) error {
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      w,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("WebUI server starting on %s\n", addr)
		errCh <- server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// CreateWebView creates a new WebView that implements dgclient.View
func CreateWebView(opts dgclient.ViewOptions) (dgclient.View, error) {
	return NewWebView(opts)
}
