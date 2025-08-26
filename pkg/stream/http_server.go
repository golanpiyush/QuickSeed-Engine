// =============================================================================
// pkg/stream/http_server.go - HTTP Streaming Server
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

// Server handles HTTP streaming with Range request support
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

// Start starts the HTTP server
func (s *Server) Start() error {
    fmt.Printf("Starting HTTP server on port %d...\n", s.port)
    
    // Clear any existing handlers to avoid conflicts
    http.DefaultServeMux = http.NewServeMux()
    
    // Register handlers with explicit routing
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
    
    // Start server in goroutine
    addr := fmt.Sprintf("127.0.0.1:%d", s.port)
    fmt.Printf("Attempting to bind to %s\n", addr)
    
    go func() {
        server := &http.Server{
            Addr: addr,
        }
        s.server = server
        
        fmt.Printf("HTTP server starting...\n")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            fmt.Printf("HTTP server failed: %v\n", err)
        }
    }()
    
    // Give server time to start
    time.Sleep(100 * time.Millisecond)
    fmt.Printf("HTTP server should be ready at http://127.0.0.1:%d\n", s.port)
    
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

// handleTest provides a simple test endpoint
func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== TEST REQUEST from %s ===\n", r.RemoteAddr)
    
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusOK)
    
    response := "QuickSeed HTTP server is working!\n"
    if s.file != nil {
        response += fmt.Sprintf("File: %s\nSize: %d bytes\n", s.file.DisplayPath(), s.file.Length())
    } else {
        response += "No file loaded\n"
    }
    
    w.Write([]byte(response))
    fmt.Printf("Test response sent successfully\n")
}

// handleRoot provides basic server info
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== ROOT REQUEST for %s from %s ===\n", r.URL.Path, r.RemoteAddr)
    
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    
    w.Header().Set("Content-Type", "text/html")
    w.WriteHeader(http.StatusOK)
    
    html := `<!DOCTYPE html>
<html>
<head><title>QuickSeed</title></head>
<body>
    <h1>QuickSeed Streaming Server</h1>
    <p>Server Status: <span style="color: green;">Running</span></p>
    <p>Available Endpoints:</p>
    <ul>
        <li><a href="/test">Test Connection</a></li>
        <li><a href="/info">File Info (JSON)</a></li>
        <li><a href="/stream">Stream Video</a></li>
    </ul>
</body>
</html>`
    
    w.Write([]byte(html))
    fmt.Printf("Root page sent successfully\n")
}

// handleInfo provides basic information about the stream
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("=== INFO REQUEST from %s ===\n", r.RemoteAddr)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    
    if s.file == nil {
        w.Write([]byte(`{"error": "No file selected", "status": "server_running"}`))
        return
    }

    json := fmt.Sprintf(`{
        "file": "%s",
        "size": %d,
        "stream_url": "%s",
        "content_type": "video/x-matroska",
        "status": "ready"
    }`, s.file.DisplayPath(), s.file.Length(), s.GetURL())
    
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
    
    // Set headers
    w.Header().Set("Content-Type", "video/x-matroska")
    w.Header().Set("Accept-Ranges", "bytes")
    w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
    
    // Handle HEAD request
    if r.Method == "HEAD" {
        fmt.Printf("HEAD request - sending headers only\n")
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
    w.WriteHeader(http.StatusOK)
    
    reader := s.file.NewReader()
    if reader == nil {
        fmt.Printf("ERROR: Could not create file reader\n")
        return
    }
    defer reader.Close()
    
    // Simple copy with progress
    buffer := make([]byte, 64*1024) // 64KB buffer
    var totalSent int64
    
    for {
        n, err := reader.Read(buffer)
        if n > 0 {
            written, writeErr := w.Write(buffer[:n])
            if writeErr != nil {
                fmt.Printf("Client disconnected: %v\n", writeErr)
                break
            }
            totalSent += int64(written)
            
            // Log progress every 10MB
            if totalSent%(10*1024*1024) == 0 {
                progress := float64(totalSent) / float64(fileSize) * 100
                fmt.Printf("Streamed %.1f%% (%d MB)\n", progress, totalSent/(1024*1024))
            }
        }
        if err != nil {
            if err == io.EOF {
                fmt.Printf("Finished streaming - EOF reached\n")
            } else {
                fmt.Printf("Read error: %v\n", err)
            }
            break
        }
    }
    
    fmt.Printf("Stream completed: %d/%d bytes sent\n", totalSent, fileSize)
}

// handleRange handles HTTP Range requests
func (s *Server) handleRange(w http.ResponseWriter, r *http.Request, fileSize int64, rangeHeader string) {
    // Parse range (basic implementation)
    rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
    parts := strings.Split(rangeHeader, "-")
    
    var start, end int64 = 0, fileSize - 1
    var err error
    
    if parts[0] != "" {
        start, err = strconv.ParseInt(parts[0], 10, 64)
        if err != nil {
            http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
            return
        }
    }
    
    if len(parts) > 1 && parts[1] != "" {
        end, err = strconv.ParseInt(parts[1], 10, 64)
        if err != nil {
            http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
            return
        }
    }
    
    if start < 0 || end >= fileSize || start > end {
        http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
        return
    }
    
    contentLength := end - start + 1
    fmt.Printf("Serving range %d-%d (%d bytes)\n", start, end, contentLength)
    
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
    
    // Copy the range
    _, err = io.CopyN(w, reader, contentLength)
    if err != nil {
        fmt.Printf("Range copy error: %v\n", err)
    } else {
        fmt.Printf("Range served successfully\n")
    }
}