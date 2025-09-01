// =============================================================================
// cmd/quickseed/main.go - CLI Application with Video Player Integration
// =============================================================================
package main

import (
    "fmt"
    "os"
    "os/exec"
    "os/signal"
    "runtime"
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
    webPlayer bool
    autoOpen  bool
)

var rootCmd = &cobra.Command{
    Use:   "quickseed [magnet-link or torrent-file]",
    Short: "QuickSeed - Fast torrent streaming engine with video player",
    Long: `QuickSeed is a high-performance torrent streaming engine that allows you to
stream video content directly from torrents without waiting for complete downloads.

Features:
- Stream videos while downloading
- Built-in web video player with controls
- Automatic largest video file selection  
- HTTP Range request support for seeking
- Multiple peer discovery methods (DHT, PEX, trackers)
- Real-time streaming statistics
- Keyboard shortcuts and fullscreen support

Examples:
  quickseed "magnet:?xt=urn:btih:..."
  quickseed movie.torrent --web-player --auto-open
  quickseed "magnet:..." --port 8080 --player vlc`,
    Args: cobra.ArbitraryArgs,
    RunE: runQuickSeed,
}

var helpCmd = &cobra.Command{
    Use:   "help",
    Short: "Display help information and available options",
    Long:  "Show detailed help information with all available command-line options in a table format",
    Run:   showHelpTable,
}

func init() {
    // Setup CLI flags
    rootCmd.Flags().IntVarP(&port, "port", "p", 8090, "HTTP server port")
    rootCmd.Flags().StringVarP(&saveDir, "save-dir", "d", "", "Download directory (default: temp)")
    rootCmd.Flags().IntVarP(&fileIndex, "file-index", "f", -1, "File index to stream (-1 for auto-select)")
    rootCmd.Flags().StringVar(&player, "player", "web", "Player to use (web|mpv|vlc|none)")
    rootCmd.Flags().IntVar(&maxPeers, "max-peers", 80, "Maximum number of peers")
    rootCmd.Flags().Int64VarP(&rateLimit, "rate-limit", "r", 0, "Download rate limit in bytes/sec (0 = unlimited)")
    rootCmd.Flags().BoolVar(&webPlayer, "web-player", true, "Use built-in web video player")
    rootCmd.Flags().BoolVar(&autoOpen, "auto-open", false, "Automatically open player when ready")
    
    // Add help command
    rootCmd.AddCommand(helpCmd)
}

func runQuickSeed(cmd *cobra.Command, args []string) error {
    // Handle special cases
    if len(args) == 0 {
        showHelpTable(cmd, args)
        return nil
    }
    
    // Check if user wants help
    if len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
        showHelpTable(cmd, args)
        return nil
    }
    
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
    
    fmt.Println("🚀 Starting QuickSeed torrent streaming engine...")
    streamURL, err := eng.Start(src, opts)
    if err != nil {
        return fmt.Errorf("failed to start engine: %v", err)
    }

    // Generate player URLs
    playerURL := fmt.Sprintf("http://127.0.0.1:%d/player", port)
    serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)

    fmt.Printf("\n✅ QuickSeed Engine Started Successfully!\n")
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    
    if webPlayer {
        fmt.Printf("🎬 Video Player:  %s\n", playerURL)
        fmt.Printf("📺 Quick Play:    %s/play\n", serverURL)
    }
    fmt.Printf("🔗 Stream URL:    %s (Not Ready Yet)\n", streamURL)
    fmt.Printf("🏠 Server Home:   %s\n", serverURL)
    fmt.Printf("📁 Download Dir:  %s\n", saveDir)
    fmt.Printf("🎯 Port:          %d\n", port)
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Monitor stream readiness and auto-open player
    if autoOpen || (webPlayer && player == "web") {
        go monitorStreamReadiness(eng, playerURL)
    }

    // Start stats monitoring
    go printStats(eng)

    fmt.Printf("\n⏳ Initializing torrent and discovering peers...\n")
    fmt.Printf("💡 Tip: You can open the video player now - it will start playing when ready!\n")
    fmt.Printf("\n🛑 Press Ctrl+C to stop\n")

    // Wait for shutdown signal
    <-sigChan
    
    fmt.Println("\n\n🛑 Shutting down QuickSeed...")
    if err := eng.Stop(); err != nil {
        fmt.Printf("Error during shutdown: %v\n", err)
    }
    fmt.Println("👋 Goodbye!")
    
    return nil
}

// showHelpTable displays all available options in a formatted table
func showHelpTable(cmd *cobra.Command, args []string) {
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("🚀 QuickSeed Engine v2.0 - Fast Torrent Video Streaming")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    
    fmt.Printf("\n📖 USAGE:\n")
    fmt.Printf("  quickseed [magnet-link or torrent-file] [OPTIONS]\n")
    fmt.Printf("  quickseed help\n\n")
    
    fmt.Printf("📋 AVAILABLE OPTIONS:\n")
    fmt.Printf("┌─────────────────────┬───────┬─────────────┬────────────────────────────────────┐\n")
    fmt.Printf("│ FLAG                │ SHORT │ DEFAULT     │ DESCRIPTION                        │\n")
    fmt.Printf("├─────────────────────┼───────┼─────────────┼────────────────────────────────────┤\n")
    fmt.Printf("│ --port              │ -p    │ 8090        │ HTTP server port                   │\n")
    fmt.Printf("│ --save-dir          │ -d    │ temp        │ Download directory                 │\n")
    fmt.Printf("│ --file-index        │ -f    │ -1          │ File index (-1 for auto-select)   │\n")
    fmt.Printf("│ --player            │       │ web         │ Player type (web/mpv/vlc/none)     │\n")
    fmt.Printf("│ --max-peers         │       │ 80          │ Maximum number of peers            │\n")
    fmt.Printf("│ --rate-limit        │ -r    │ 0           │ Download rate limit (bytes/sec)    │\n")
    fmt.Printf("│ --web-player        │       │ true        │ Enable built-in web player         │\n")
    fmt.Printf("│ --auto-open         │       │ false       │ Auto-open player when ready        │\n")
    fmt.Printf("│ --help              │ -h    │             │ Show this help message             │\n")
    fmt.Printf("└─────────────────────┴───────┴─────────────┴────────────────────────────────────┘\n")
    
    fmt.Printf("\n🎮 PLAYER OPTIONS:\n")
    fmt.Printf("┌─────────┬──────────────────────────────────────────────────────────┐\n")
    fmt.Printf("│ VALUE   │ DESCRIPTION                                              │\n")
    fmt.Printf("├─────────┼──────────────────────────────────────────────────────────┤\n")
    fmt.Printf("│ web     │ Built-in HTML5 video player with controls (recommended) │\n")
    fmt.Printf("│ mpv     │ Launch MPV media player                                  │\n")
    fmt.Printf("│ vlc     │ Launch VLC media player                                  │\n")
    fmt.Printf("│ none    │ No player, stream URL only                               │\n")
    fmt.Printf("└─────────┴──────────────────────────────────────────────────────────┘\n")
    
    fmt.Printf("\n💡 USAGE EXAMPLES:\n")
    fmt.Printf("  # Basic usage with web player (recommended)\n")
    fmt.Printf("  quickseed \"magnet:?xt=urn:btih:...\"\n\n")
    
    fmt.Printf("  # Auto-open browser when ready\n")
    fmt.Printf("  quickseed movie.torrent --auto-open\n\n")
    
    fmt.Printf("  # Use VLC player on custom port\n")
    fmt.Printf("  quickseed \"magnet:...\" --player vlc --port 9090\n\n")
    
    fmt.Printf("  # Custom download directory with rate limiting\n")
    fmt.Printf("  quickseed movie.torrent --save-dir ./downloads --rate-limit 1048576\n\n")
    
    fmt.Printf("  # Select specific file and disable web player\n")
    fmt.Printf("  quickseed \"magnet:...\" --file-index 2 --web-player=false --player mpv\n\n")
    
    fmt.Printf("🔗 WEB INTERFACE:\n")
    fmt.Printf("  After starting, access these URLs in your browser:\n")
    fmt.Printf("  • Video Player:  http://127.0.0.1:[PORT]/player\n")
    fmt.Printf("  • Quick Play:    http://127.0.0.1:[PORT]/play\n")
    fmt.Printf("  • Server Info:   http://127.0.0.1:[PORT]/\n")
    fmt.Printf("  • Stream URL:    http://127.0.0.1:[PORT]/stream\n\n")
    
    fmt.Printf("⚡ FEATURES:\n")
    fmt.Printf("  • Stream videos while downloading (no waiting for completion)\n")
    fmt.Printf("  • HTML5 video player with seek, fullscreen, keyboard shortcuts\n")
    fmt.Printf("  • Automatic largest video file detection\n")
    fmt.Printf("  • HTTP Range request support for smooth seeking\n")
    fmt.Printf("  • Real-time download statistics and progress\n")
    fmt.Printf("  • Cross-platform browser integration\n\n")
    
    fmt.Printf("🎹 KEYBOARD SHORTCUTS (Web Player):\n")
    fmt.Printf("  Space    - Play/Pause\n")
    fmt.Printf("  F        - Fullscreen toggle\n")
    fmt.Printf("  R        - Restart video\n")
    fmt.Printf("  ← →      - Seek ±10 seconds\n")
    fmt.Printf("  ↑ ↓      - Volume up/down\n")
    fmt.Printf("  M        - Mute/unmute\n\n")
    
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
func monitorStreamReadiness(eng api.Engine, playerURL string) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    hasOpened := false
    
    for range ticker.C {
        stats := eng.Stats()
        
        if stats.StreamReady && !hasOpened {
            hasOpened = true
            fmt.Printf("\n\n🎉 Stream is now ready! Opening video player...\n")
            
            if err := openBrowser(playerURL); err != nil {
                fmt.Printf("⚠️  Could not auto-open browser: %v\n", err)
                fmt.Printf("🌐 Please manually open: %s\n", playerURL)
            } else {
                fmt.Printf("🌐 Video player opened in your browser!\n")
            }
            
            fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
            break
        }
    }
}

// openBrowser opens the given URL in the default browser
func openBrowser(url string) error {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "windows":
        cmd = "rundll32"
        args = []string{"url.dll,FileProtocolHandler", url}
    case "darwin":
        cmd = "open"
        args = []string{url}
    default: // linux and others
        cmd = "xdg-open"
        args = []string{url}
    }

    return exec.Command(cmd, args...).Start()
}

// printStats periodically displays streaming statistics
func printStats(eng api.Engine) {
    ticker := time.NewTicker(3 * time.Second)
    defer ticker.Stop()

    lastPrint := time.Now()
    
    for range ticker.C {
        stats := eng.Stats()
        
        // Clear the current line and print updated stats
        fmt.Printf("\r\033[K") // Clear line
        
        status := "🔴 Initializing"
        if stats.StreamReady {
            status = "🟢 Ready to Stream"
        } else if stats.Progress > 0 {
            status = "🟡 Buffering"
        }
        
        fmt.Printf("📊 %s | %.1f%% | ⬇️%.1fMB/s ⬆️%.1fMB/s | 👥%d peers",
            status,
            stats.Progress*100,
            stats.DownloadSpeed/1024/1024,
            stats.UploadSpeed/1024/1024,
            stats.Peers)
            
        // Print detailed status every 15 seconds
        if time.Since(lastPrint) >= 15*time.Second {
            fmt.Printf("\n")
            printDetailedStatus(stats)
            lastPrint = time.Now()
        }
    }
}

// printDetailedStatus prints comprehensive streaming information
func printDetailedStatus(stats api.Stats) {
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    fmt.Printf("📈 Detailed Status Report:\n")
    fmt.Printf("   Progress:      %.2f%%\n", stats.Progress*100)
    fmt.Printf("   Download Rate: %.2f MB/s\n", stats.DownloadSpeed/1024/1024)
    fmt.Printf("   Upload Rate:   %.2f MB/s\n", stats.UploadSpeed/1024/1024)
    fmt.Printf("   Connected Peers: %d\n", stats.Peers)
    fmt.Printf("   Stream Status: %s\n", formatStreamStatus(stats.StreamReady))
    
    if stats.StreamReady {
        fmt.Printf("   🎬 You can now play the video in your browser!\n")
    } else {
        fmt.Printf("   ⏳ Stream will be ready soon...\n")
    }
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func formatStreamStatus(ready bool) string {
    if ready {
        return "✅ Ready - Video can be played"
    }
    return "⏳ Buffering initial content"
}

// launchExternalPlayer launches external video players
func launchExternalPlayer(playerType, streamURL string) error {
    var cmd *exec.Cmd
    
    switch playerType {
    case "vlc":
        cmd = exec.Command("vlc", streamURL)
    case "mpv":
        cmd = exec.Command("mpv", streamURL)
    default:
        return fmt.Errorf("unsupported player: %s", playerType)
    }
    
    fmt.Printf("🎬 Launching %s player...\n", playerType)
    return cmd.Start()
}

func main() {
    // Add version info
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("🚀 QuickSeed Engine v2.0 - Fast Torrent Video Streaming")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
        os.Exit(1)
    }
}