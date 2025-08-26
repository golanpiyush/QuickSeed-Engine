// =============================================================================
// pkg/engine/torrent_engine.go - Main Torrent Engine
// =============================================================================
package engine

import (
    "fmt"
    "os/exec"
    "time"
    
    "github.com/anacrolix/torrent"
    
    "quickseed/pkg/api"
    "quickseed/pkg/stream"
    "quickseed/pkg/utils"
)

// TorrentEngine implements the Engine interface
type TorrentEngine struct {
    client    *torrent.Client
    torrent   *torrent.Torrent
    file      *torrent.File
    server    *stream.Server
    opts      api.Options
    startTime time.Time
    running   bool
}

// NewTorrentEngine creates a new torrent engine instance
func NewTorrentEngine() *TorrentEngine {
    return &TorrentEngine{}
}

// Start initializes and starts the torrent streaming engine
func (e *TorrentEngine) Start(src string, opts api.Options) (string, error) {
    e.opts = opts
    e.startTime = time.Now()
    e.running = true

    // Create torrent client configuration
    cfg := torrent.NewDefaultClientConfig()
    cfg.DataDir = opts.SaveDir
    cfg.DisablePEX = false  // Enable PEX for peer discovery
    cfg.NoDHT = false       // Enable DHT for peer discovery
    cfg.DisableTrackers = false // Enable trackers
    
    if opts.MaxPeers > 0 {
        cfg.EstablishedConnsPerTorrent = opts.MaxPeers
    }

    // Create torrent client
    client, err := torrent.NewClient(cfg)
    if err != nil {
        return "", fmt.Errorf("failed to create torrent client: %v", err)
    }
    e.client = client

    // Add torrent (support both magnet links and .torrent files)
    var t *torrent.Torrent
    if utils.FileExists(src) {
        // .torrent file
        t, err = client.AddTorrentFromFile(src)
    } else {
        // Magnet link
        t, err = client.AddMagnet(src)
    }
    
    if err != nil {
        return "", fmt.Errorf("failed to add torrent: %v", err)
    }
    e.torrent = t

    // Wait for torrent info
    fmt.Println("Getting torrent info...")
    <-t.GotInfo()

    // Select file to stream
    if opts.FileIndex >= 0 && opts.FileIndex < len(t.Files()) {
        e.file = t.Files()[opts.FileIndex]
    } else {
        // Auto-select largest video file
        file, _ := utils.FindLargestVideoFile(t)
        if file == nil {
            return "", fmt.Errorf("no video files found in torrent")
        }
        e.file = file
    }

    fmt.Printf("Selected file: %s (%.2f MB)\n", 
        e.file.DisplayPath(), 
        float64(e.file.Length())/1024/1024)

    // Enable sequential downloading for the selected file
    e.file.SetPriority(torrent.PiecePriorityNow)
    
    // Download file pieces sequentially
    go e.enableSequentialDownload()

    // Setup HTTP streaming server
    e.server = stream.NewServer(opts.Port)
    e.server.SetFile(e.file)
    
    if err := e.server.Start(); err != nil {
        return "", fmt.Errorf("failed to start HTTP server: %v", err)
    }

    streamURL := e.server.GetURL()
    fmt.Printf("Streaming URL: %s\n", streamURL)

    // Launch player if specified
    if opts.Player != "none" && opts.Player != "" {
        go e.launchPlayer(streamURL, opts.Player)
    }

    return streamURL, nil
}

// SelectFile allows changing the file being streamed
func (e *TorrentEngine) SelectFile(index int) error {
    if e.torrent == nil {
        return fmt.Errorf("no torrent loaded")
    }

    files := e.torrent.Files()
    if index < 0 || index >= len(files) {
        return fmt.Errorf("file index %d out of range (0-%d)", index, len(files)-1)
    }

    e.file = files[index]
    e.server.SetFile(e.file)
    
    // Set priority for new file
    e.file.SetPriority(torrent.PiecePriorityNow)
    go e.enableSequentialDownload()

    fmt.Printf("Switched to file: %s\n", e.file.DisplayPath())
    return nil
}

// Stats returns current engine statistics
func (e *TorrentEngine) Stats() api.Stats {
    if e.torrent == nil {
        return api.Stats{}
    }

    stats := e.torrent.Stats()
    
    // Calculate bytes completed manually if needed
    var bytesCompleted int64
    if e.torrent.Info() != nil {
        bytesCompleted = e.torrent.BytesCompleted()
    }
    
    return api.Stats{
        TorrentName:     e.torrent.Name(),
        TotalSize:       e.torrent.Length(),
        Downloaded:      bytesCompleted,
        DownloadSpeed:   float64(stats.BytesReadData.Int64()),
        UploadSpeed:     float64(stats.BytesWrittenData.Int64()),
        Progress:        float64(bytesCompleted) / float64(e.torrent.Length()),
        Peers:           stats.ActivePeers,
        Seeders:         stats.ConnectedSeeders,
        StreamingFile:   e.file.DisplayPath(),
        StreamingSize:   e.file.Length(),
        StreamReady:     e.isStreamReady(),
        Uptime:          time.Since(e.startTime),
    }
}

// Stop gracefully shuts down the engine
func (e *TorrentEngine) Stop() error {
    e.running = false
    
    if e.server != nil {
        e.server.Stop()
    }
    
    if e.torrent != nil {
        e.torrent.Drop()
    }
    
    if e.client != nil {
        e.client.Close()
    }
    
    return nil
}

// enableSequentialDownload ensures pieces are downloaded in order for streaming
func (e *TorrentEngine) enableSequentialDownload() {
    if e.file == nil {
        return
    }

    // Set high priority for first few pieces to start streaming quickly
    torr := e.file.Torrent()
    numPieces := torr.NumPieces()
    
    // Prioritize first 10 pieces or 5% of total pieces, whichever is smaller
    piecesToPrioritize := 10
    if numPieces < 200 {
        piecesToPrioritize = numPieces / 20 // 5% of pieces
        if piecesToPrioritize < 3 {
            piecesToPrioritize = 3
        }
    }
    
    for i := 0; i < piecesToPrioritize && i < numPieces; i++ {
        torr.Piece(i).SetPriority(torrent.PiecePriorityNow)
    }
}

// isStreamReady checks if enough data is available to start streaming
func (e *TorrentEngine) isStreamReady() bool {
    if e.file == nil {
        return false
    }
    
    // Consider ready if first 5% is downloaded
    firstPieces := int(float64(e.file.Torrent().NumPieces()) * 0.05)
    if firstPieces < 5 {
        firstPieces = 5
    }
    
    completed := 0
    for i := 0; i < firstPieces && i < e.file.Torrent().NumPieces(); i++ {
        if e.file.Torrent().Piece(i).State().Complete {
            completed++
        }
    }
    
    return float64(completed)/float64(firstPieces) > 0.5
}

// launchPlayer attempts to launch the specified video player
func (e *TorrentEngine) launchPlayer(url, player string) {
    // Wait a bit for streaming to become ready
    time.Sleep(3 * time.Second)

    var cmd *exec.Cmd
    
    switch player {
    case "mpv":
        cmd = exec.Command("mpv", url)
    case "vlc":
        cmd = exec.Command("vlc", url)
    default:
        fmt.Printf("Unknown player: %s\n", player)
        return
    }

    fmt.Printf("Launching %s...\n", player)
    if err := cmd.Start(); err != nil {
        fmt.Printf("Failed to launch %s: %v\n", player, err)
        fmt.Printf("Please manually open: %s\n", url)
    }
}