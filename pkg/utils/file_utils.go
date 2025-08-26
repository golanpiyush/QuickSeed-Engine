// =============================================================================
// pkg/utils/file_utils.go - File Utilities
// =============================================================================
package utils

import (
    "os"
    "path/filepath"
    "strings"
    
    "github.com/anacrolix/torrent"
)

// VideoExtensions contains common video file extensions
var VideoExtensions = map[string]bool{
    ".mp4":  true,
    ".mkv":  true,
    ".avi":  true,
    ".mov":  true,
    ".wmv":  true,
    ".flv":  true,
    ".webm": true,
    ".m4v":  true,
    ".3gp":  true,
    ".ts":   true,
    ".m2ts": true,
}

// IsVideoFile checks if a file is a video file based on extension
func IsVideoFile(filename string) bool {
    ext := strings.ToLower(filepath.Ext(filename))
    return VideoExtensions[ext]
}

// FindLargestVideoFile finds the largest video file in the torrent
func FindLargestVideoFile(t *torrent.Torrent) (*torrent.File, int) {
    var largestFile *torrent.File
    var largestIndex int
    var largestSize int64

    for i, file := range t.Files() {
        if IsVideoFile(file.DisplayPath()) && file.Length() > largestSize {
            largestFile = file
            largestIndex = i
            largestSize = file.Length()
        }
    }

    return largestFile, largestIndex
}

// CreateTempDir creates a temporary directory for downloads
func CreateTempDir() (string, error) {
    return os.MkdirTemp("", "quickseed-*")
}

// FileExists checks if a file exists
func FileExists(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}