// =============================================================================
// pkg/stream/http_server.go - HTTP Streaming Server with Video Player UI
// =============================================================================
package stream

import (
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"
    
    "github.com/anacrolix/torrent"
)

// Server handles HTTP streaming with Range request support and video player UI
type Server struct {
    port   int
    file   *torrent.File
    server *http.Server
}

// NewServer creates a new streaming server
func NewServer(port int) *Server {
    return &Server{
        port: port,
    }
}

// SetFile sets the file to be streamed
func (s *Server) SetFile(file *torrent.File) {
    s.file = file
}

// Start starts the HTTP server with all endpoints including video player
func (s *Server) Start() error {
    fmt.Printf("Starting HTTP server on port %d...\n", s.port)
    
    // Clear any existing handlers to avoid conflicts
    http.DefaultServeMux = http.NewServeMux()
    
    // Register all handlers with explicit routing
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("Request: %s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)
        if r.URL.Path == "/" {
            s.handleRoot(w, r)
        } else if r.URL.Path == "/favicon.ico" {
            // Handle favicon requests gracefully
            http.NotFound(w, r)
        } else {
            http.NotFound(w, r)
        }
    })
    
    http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("Test request from %s\n", r.RemoteAddr)
        s.handleTest(w, r)
    })
    
    http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("Info request from %s\n", r.RemoteAddr)
        s.handleInfo(w, r)
    })
    
    http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("Stream request from %s\n", r.RemoteAddr)
        s.handleStream(w, r)
    })
    
    // NEW: Video Player UI endpoint
    http.HandleFunc("/player", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("Player UI request from %s\n", r.RemoteAddr)
        s.handlePlayer(w, r)
    })
    
    // NEW: Direct play endpoint (redirects to player)
    http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("Play redirect from %s\n", r.RemoteAddr)
        http.Redirect(w, r, "/player", http.StatusFound)
    })
    
    // Start server in goroutine
    addr := fmt.Sprintf("127.0.0.1:%d", s.port)
    fmt.Printf("Attempting to bind to %s\n", addr)
    
    go func() {
        server := &http.Server{
            Addr: addr,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 0, // No timeout for streaming
            IdleTimeout:  120 * time.Second,
        }
        s.server = server
        
        fmt.Printf("HTTP server starting...\n")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            fmt.Printf("HTTP server failed: %v\n", err)
        }
    }()
    
    // Give server time to start
    time.Sleep(100 * time.Millisecond)
    fmt.Printf("HTTP server ready at http://127.0.0.1:%d\n", s.port)
    fmt.Printf("🎬 Video Player: http://127.0.0.1:%d/player\n", s.port)
    fmt.Printf("📺 Direct Play: http://127.0.0.1:%d/play\n", s.port)
    
    return nil
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
    if s.server != nil {
        return s.server.Close()
    }
    return nil
}

// GetURL returns the streaming URL
func (s *Server) GetURL() string {
    return fmt.Sprintf("http://127.0.0.1:%d/stream", s.port)
}

// GetPlayerURL returns the video player URL
func (s *Server) GetPlayerURL() string {
    return fmt.Sprintf("http://127.0.0.1:%d/player", s.port)
}

// handleTest provides a simple test endpoint
func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== TEST REQUEST from %s ===\n", r.RemoteAddr)
    
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusOK)
    
    response := "QuickSeed HTTP server is working!\n"
    if s.file != nil {
        response += fmt.Sprintf("File: %s\nSize: %d bytes\n", s.file.DisplayPath(), s.file.Length())
        response += fmt.Sprintf("Video Player: http://127.0.0.1:%d/player\n", s.port)
    } else {
        response += "No file loaded\n"
    }
    
    w.Write([]byte(response))
    fmt.Printf("Test response sent successfully\n")
}

// handleRoot provides basic server info with video player links
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== ROOT REQUEST for %s from %s ===\n", r.URL.Path, r.RemoteAddr)
    
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    
    w.Header().Set("Content-Type", "text/html")
    w.WriteHeader(http.StatusOK)
    
    playerStatus := "No file loaded"
    playerButton := `<li><span style="color: #666;">Video Player (No file loaded)</span></li>`
    
    if s.file != nil {
        playerStatus = fmt.Sprintf("Ready to stream: %s", s.file.DisplayPath())
        playerButton = fmt.Sprintf(`<li><a href="/player" style="color: #e74c3c; font-weight: bold;">🎬 Launch Video Player</a></li>
        <li><a href="/play" style="color: #e67e22;">📺 Quick Play</a></li>`)
    }
    
    html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>QuickSeed Server</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; 
            max-width: 800px; 
            margin: 40px auto; 
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            border-radius: 10px;
        }
        h1 { color: white; text-shadow: 2px 2px 4px rgba(0,0,0,0.5); }
        .status { 
            padding: 15px; 
            background: rgba(255,255,255,0.1); 
            border-radius: 8px; 
            margin: 20px 0;
            backdrop-filter: blur(10px);
        }
        ul { list-style: none; padding: 0; }
        li { 
            padding: 10px; 
            margin: 8px 0; 
            background: rgba(255,255,255,0.1); 
            border-radius: 6px;
            border-left: 4px solid #3498db;
        }
        a { 
            color: #74b9ff; 
            text-decoration: none; 
            font-weight: 500;
        }
        a:hover { 
            color: white; 
            text-shadow: 0 0 10px rgba(255,255,255,0.8);
        }
        .highlight {
            border-left-color: #e74c3c !important;
            background: rgba(231, 76, 60, 0.2);
        }
    </style>
</head>
<body>
    <h1>🚀 QuickSeed Streaming Server</h1>
    <div class="status">
        <p><strong>Server Status:</strong> <span style="color: #2ecc71;">✅ Running</span></p>
        <p><strong>Player Status:</strong> %s</p>
    </div>
    
    <h2>Available Endpoints:</h2>
    <ul>
        %s
        <li><a href="/test">🔧 Test Connection</a></li>
        <li><a href="/info">📊 File Info (JSON)</a></li>
        <li><a href="/stream">🎥 Direct Stream URL</a></li>
    </ul>
    
    <div style="margin-top: 30px; padding: 15px; background: rgba(0,0,0,0.2); border-radius: 8px; font-size: 0.9em;">
        <p><strong>💡 Tips:</strong></p>
        <ul style="margin-left: 20px;">
            <li>• Use the Video Player for the best streaming experience</li>
            <li>• Player supports fullscreen, keyboard shortcuts, and download</li>
            <li>• Stream URL can be used in external players like VLC</li>
        </ul>
    </div>
</body>
</html>`, playerStatus, playerButton)
    
    w.Write([]byte(html))
    fmt.Printf("Root page sent successfully\n")
}

// handleInfo provides information about the stream with player URLs
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== INFO REQUEST from %s ===\n", r.RemoteAddr)
    
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(http.StatusOK)
    
    if s.file == nil {
        w.Write([]byte(`{
            "error": "No file selected", 
            "status": "server_running",
            "player_url": "` + s.GetPlayerURL() + `",
            "endpoints": {
                "stream": "/stream",
                "player": "/player", 
                "info": "/info",
                "test": "/test"
            }
        }`))
        return
    }

    json := fmt.Sprintf(`{
        "file": "%s",
        "size": %d,
        "size_formatted": "%s",
        "stream_url": "%s",
        "player_url": "%s",
        "content_type": "video/x-matroska",
        "status": "ready",
        "server_port": %d,
        "endpoints": {
            "stream": "/stream",
            "player": "/player",
            "info": "/info", 
            "test": "/test",
            "play": "/play"
        },
        "features": {
            "range_requests": true,
            "sequential_download": true,
            "web_player": true,
            "keyboard_shortcuts": true
        }
    }`, 
        strings.Replace(s.file.DisplayPath(), `\`, `/`, -1), 
        s.file.Length(),
        formatBytes(s.file.Length()),
        s.GetURL(), 
        s.GetPlayerURL(),
        s.port)
    
    w.Write([]byte(json))
    fmt.Printf("Info response sent successfully\n")
}

// handleStream handles the main streaming endpoint
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== STREAM REQUEST ===\n")
    fmt.Printf("Method: %s\n", r.Method)
    fmt.Printf("Remote: %s\n", r.RemoteAddr)
    fmt.Printf("User-Agent: %s\n", r.Header.Get("User-Agent"))
    fmt.Printf("Range: %s\n", r.Header.Get("Range"))
    
    if s.file == nil {
        fmt.Printf("ERROR: No file available for streaming\n")
        http.Error(w, "No file available", http.StatusServiceUnavailable)
        return
    }

    // Only allow GET and HEAD
    if r.Method != "GET" && r.Method != "HEAD" {
        fmt.Printf("ERROR: Method %s not allowed\n", r.Method)
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    fileSize := s.file.Length()
    fmt.Printf("Streaming file: %s (%d bytes)\n", s.file.DisplayPath(), fileSize)
    
    // Set CORS headers for web player
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
    w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
    
    // Set streaming headers
    w.Header().Set("Content-Type", "video/x-matroska")
    w.Header().Set("Accept-Ranges", "bytes")
    w.Header().Set("Cache-Control", "no-cache")
    
    // Handle OPTIONS request for CORS
    if r.Method == "OPTIONS" {
        fmt.Printf("OPTIONS request - sending CORS headers\n")
        w.WriteHeader(http.StatusOK)
        return
    }
    
    // Handle HEAD request
    if r.Method == "HEAD" {
        fmt.Printf("HEAD request - sending headers only\n")
        w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
        w.WriteHeader(http.StatusOK)
        return
    }
    
    // Handle Range request
    rangeHeader := r.Header.Get("Range")
    if rangeHeader != "" {
        fmt.Printf("Range request: %s\n", rangeHeader)
        s.handleRange(w, r, fileSize, rangeHeader)
        return
    }
    
    // Full file request
    fmt.Printf("Full file request - starting stream\n")
    w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
    w.WriteHeader(http.StatusOK)
    
    reader := s.file.NewReader()
    if reader == nil {
        fmt.Printf("ERROR: Could not create file reader\n")
        return
    }
    defer reader.Close()
    
    // Optimized streaming with larger buffer
    buffer := make([]byte, 128*1024) // 128KB buffer for better streaming
    var totalSent int64
    lastProgressLog := int64(0)
    
    for {
        n, err := reader.Read(buffer)
        if n > 0 {
            written, writeErr := w.Write(buffer[:n])
            if writeErr != nil {
                fmt.Printf("Client disconnected: %v\n", writeErr)
                break
            }
            totalSent += int64(written)
            
            // Log progress every 5MB
            if totalSent-lastProgressLog >= 5*1024*1024 {
                progress := float64(totalSent) / float64(fileSize) * 100
                fmt.Printf("Streaming progress: %.1f%% (%s/%s)\n", 
                    progress, 
                    formatBytes(totalSent), 
                    formatBytes(fileSize))
                lastProgressLog = totalSent
                
                // Flush data to client
                if flusher, ok := w.(http.Flusher); ok {
                    flusher.Flush()
                }
            }
        }
        if err != nil {
            if err == io.EOF {
                fmt.Printf("Streaming completed successfully - EOF reached\n")
            } else {
                fmt.Printf("Read error: %v\n", err)
            }
            break
        }
    }
    
    fmt.Printf("Stream finished: %s/%s sent\n", formatBytes(totalSent), formatBytes(fileSize))
}

// handleRange handles HTTP Range requests with improved error handling
func (s *Server) handleRange(w http.ResponseWriter, r *http.Request, fileSize int64, rangeHeader string) {
    // Parse range (basic implementation)
    rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
    parts := strings.Split(rangeHeader, "-")
    
    var start, end int64 = 0, fileSize - 1
    var err error
    
    if parts[0] != "" {
        start, err = strconv.ParseInt(parts[0], 10, 64)
        if err != nil {
            fmt.Printf("Invalid range start: %s\n", parts[0])
            http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
            return
        }
    }
    
    if len(parts) > 1 && parts[1] != "" {
        end, err = strconv.ParseInt(parts[1], 10, 64)
        if err != nil {
            fmt.Printf("Invalid range end: %s\n", parts[1])
            http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
            return
        }
    }
    
    if start < 0 || end >= fileSize || start > end {
        fmt.Printf("Range out of bounds: %d-%d (file size: %d)\n", start, end, fileSize)
        w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
        http.Error(w, "Requested range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
        return
    }
    
    contentLength := end - start + 1
    fmt.Printf("Serving range %d-%d (%s)\n", start, end, formatBytes(contentLength))
    
    w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
    w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
    w.WriteHeader(http.StatusPartialContent)
    
    // Get reader and seek
    reader := s.file.NewReader()
    if reader == nil {
        fmt.Printf("ERROR: Could not create reader for range\n")
        return
    }
    defer reader.Close()
    
    _, err = reader.Seek(start, 0)
    if err != nil {
        fmt.Printf("Seek error: %v\n", err)
        return
    }
    
    // Copy the range with buffering
    buffer := make([]byte, 64*1024) // 64KB buffer
    remaining := contentLength
    
    for remaining > 0 {
        toRead := int64(len(buffer))
        if remaining < toRead {
            toRead = remaining
        }
        
        n, readErr := reader.Read(buffer[:toRead])
        if n > 0 {
            written, writeErr := w.Write(buffer[:n])
            if writeErr != nil {
                fmt.Printf("Range write error: %v\n", writeErr)
                break
            }
            remaining -= int64(written)
        }
        
        if readErr != nil {
            if readErr == io.EOF {
                break
            }
            fmt.Printf("Range read error: %v\n", readErr)
            break
        }
    }
    
    if remaining == 0 {
        fmt.Printf("Range served successfully\n")
    } else {
        fmt.Printf("Range partially served: %d bytes remaining\n", remaining)
    }
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
    if bytes < 1024 {
        return fmt.Sprintf("%d B", bytes)
    }
    if bytes < 1024*1024 {
        return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
    }
    if bytes < 1024*1024*1024 {
        return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
    }
    return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}