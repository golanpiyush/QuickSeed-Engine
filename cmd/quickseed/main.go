// =============================================================================
// cmd/quickseed/main.go - CLI Application
// =============================================================================
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/spf13/cobra"
    
    "quickseed/pkg/api"
    "quickseed/pkg/engine"
    "quickseed/pkg/utils"
)

var (
    port      int
    saveDir   string
    fileIndex int
    player    string
    maxPeers  int
    rateLimit int64
)

var rootCmd = &cobra.Command{
    Use:   "quickseed [magnet-link or torrent-file]",
    Short: "QuickSeed - Fast torrent streaming engine",
    Long: `QuickSeed is a high-performance torrent streaming engine that allows you to
stream video content directly from torrents without waiting for complete downloads.

Features:
- Stream videos while downloading
- Automatic largest video file selection  
- HTTP Range request support for seeking
- Multiple peer discovery methods (DHT, PEX, trackers)
- Built-in video player integration
- Real-time streaming statistics`,
    Args: cobra.ExactArgs(1),
    RunE: runQuickSeed,
}

func init() {
    // Setup CLI flags
    rootCmd.Flags().IntVarP(&port, "port", "p", 8090, "HTTP server port")
    rootCmd.Flags().StringVarP(&saveDir, "save-dir", "d", "", "Download directory (default: temp)")
    rootCmd.Flags().IntVarP(&fileIndex, "file-index", "f", -1, "File index to stream (-1 for auto-select)")
    rootCmd.Flags().StringVar(&player, "player", "none", "Auto-launch player (mpv|vlc|none)")
    rootCmd.Flags().IntVar(&maxPeers, "max-peers", 80, "Maximum number of peers")
    rootCmd.Flags().Int64VarP(&rateLimit, "rate-limit", "r", 0, "Download rate limit in bytes/sec (0 = unlimited)")
}

func runQuickSeed(cmd *cobra.Command, args []string) error {
    src := args[0]
    
    // Setup save directory
    if saveDir == "" {
        var err error
        saveDir, err = utils.CreateTempDir()
        if err != nil {
            return fmt.Errorf("failed to create temp directory: %v", err)
        }
        fmt.Printf("Using temp directory: %s\n", saveDir)
    }

    // Create engine options
    opts := api.Options{
        Port:      port,
        SaveDir:   saveDir,
        FileIndex: fileIndex,
        Player:    player,
        MaxPeers:  maxPeers,
        RateLimit: rateLimit,
    }

    // Create and start engine
    eng := engine.NewTorrentEngine()
    
    fmt.Println("Starting QuickSeed torrent streaming engine...")
    streamURL, err := eng.Start(src, opts)
    if err != nil {
        return fmt.Errorf("failed to start engine: %v", err)
    }

    fmt.Printf("\n✅ QuickSeed initialised!\n")
    fmt.Printf("🎬 Stream URL (Not Ready): %s\n", streamURL)
    fmt.Printf("📁 Download Dir: %s\n", saveDir)
    fmt.Printf("🎯 Press Ctrl+C to stop\n\n")

    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start stats monitoring
    go printStats(eng)

    // Wait for shutdown signal
    <-sigChan
    
    fmt.Println("\n🛑 Shutting down QuickSeed...")
    if err := eng.Stop(); err != nil {
        fmt.Printf("Error during shutdown: %v\n", err)
    }
    fmt.Println("👋 Goodbye!")
    
    return nil
}

// printStats periodically displays streaming statistics
func printStats(eng api.Engine) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := eng.Stats()
        
        fmt.Printf("\r📊 Progress: %.1f%% | Speed: %.1fMb/s %.1fMb/s | Peers Discoverd: %d | %s",
            stats.Progress*100,
            stats.DownloadSpeed/1024/1024,
            stats.UploadSpeed/1024/1024,
            stats.Peers,
            formatStreamStatus(stats.StreamReady))
    }
}

func formatStreamStatus(ready bool) string {
    if ready {
        return "State: 🟢 Stream Ready"
    }
    return "State: 🟡 Buffering..."
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}