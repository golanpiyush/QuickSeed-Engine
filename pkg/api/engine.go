// =============================================================================
// pkg/api/engine.go - Engine API Interface
// =============================================================================
package api

import "time"

// Engine provides the main interface for the torrent streaming engine
type Engine interface {
    Start(src string, opts Options) (url string, err error)
    SelectFile(index int) error
    Stats() Stats
    Stop() error
}


// Options configures the engine behavior
type Options struct {
    Port        int    // HTTP server port
    SaveDir     string // Download directory
    FileIndex   int    // Specific file to stream (-1 for auto-select)
    Player      string // Player to launch (mpv|vlc|none)
    MaxPeers    int    // Maximum number of peers
    RateLimit   int64  // Download rate limit in bytes/sec (0 = unlimited)
}

// Stats provides runtime statistics
type Stats struct {
    TorrentName     string        // Name of the torrent
    TotalSize       int64         // Total size in bytes
    Downloaded      int64         // Downloaded bytes
    DownloadSpeed   float64       // Download speed in bytes/sec
    UploadSpeed     float64       // Upload speed in bytes/sec
    Progress        float64       // Download progress (0-1)
    Peers           int           // Number of connected peers
    Seeders         int           // Number of seeders
    StreamingFile   string        // Currently streaming file name
    StreamingSize   int64         // Size of streaming file
    StreamReady     bool          // Whether streaming is ready
    Uptime          time.Duration // Engine uptime
}
