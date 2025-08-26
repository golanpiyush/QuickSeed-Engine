# 🎬 QuickSeed - Torrent Streaming Engine

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![GitHub Issues](https://img.shields.io/github/issues/golanpiyush/QuickSeed-Engine)](https://github.com/golanpiyush/QuickSeed-Engine/issues)
[![GitHub Stars](https://img.shields.io/github/stars/golanpiyush/QuickSeed-Engine)](https://github.com/golanpiyush/QuickSeed-Engine/stargazers)

**QuickSeed** is a fast and lightweight torrent streaming engine written in Go.  
It allows you to stream videos directly from torrents **without waiting for full downloads**.

---

## ✨ Features
- 🚀 **Instant streaming** while downloading  
- 📺 **HTTP server with Range support** (seek like Netflix)  
- 🔍 **Automatic video selection** (largest file or by index)  
- 🌍 **Peer discovery** (DHT, PEX, Trackers enabled)  
- 🎛️ **CLI control** with Cobra  
- 🎬 **Auto-launch video players** (`mpv`, `vlc`)  
- 📊 **Real-time streaming stats** in terminal  
- ⚡ **Lightweight**: minimal dependencies, pure Go core  
- 🔧 **Configurable**: rate-limiting, peer control, save directory  
- 🔒 **Secure by default**: sandboxed streaming, no external trackers unless specified  

---

## 📦 Installation
```bash
git clone https://github.com/golanpiyush/QuickSeed-Engine.git
cd QuickSeed-Engine
go build -o quickseed.exe ./cmd/quickseed
```

This creates a **`quickseed`** (Linux/macOS) or **`quickseed.exe`** (Windows) binary.

---

## ▶️ Usage
```bash
./quickseed "path/to/file.torrent"
./quickseed "magnet:?xt=urn:btih:..."
```

After starting, open your browser or media player at:

```
http://localhost:8090
```

Example with **mpv**:
```bash
mpv http://localhost:8090
```

Example with **VLC**:
```bash
vlc http://localhost:8090
```

---

## ⚙️ Options
| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port` | `8090` | HTTP server port |
| `-d, --save-dir` | `temp` | Download directory |
| `-f, --file-index` | `-1` | File index to stream (-1 = auto) |
| `--player` | `none` | Auto launch player (`mpv`, `vlc`, `none`) |
| `--max-peers` | `80` | Maximum peer connections |
| `-r, --rate-limit` | `0` | Download speed limit (bytes/sec, 0 = unlimited) |
| `--no-dht` | `false` | Disable DHT peer discovery |
| `--no-pex` | `false` | Disable peer exchange |

---

## 🏗️ Architecture

QuickSeed is built with a **modular architecture**:

```
[ Torrent Input ] → [ Torrent Engine (anacrolix/torrent) ] → [ Piece Selection ]
        ↓                         ↓
   [ DHT / Trackers ]      [ Disk Cache + Memory Buffer ]
        ↓                         ↓
    [ Peer Connections ] → [ HTTP Range Server ] → [ Media Player ]
```

- **Torrent Input**: Accepts `.torrent` files or magnet links  
- **Torrent Engine**: Manages peers, trackers, piece requests  
- **Piece Selection**: Smart streaming mode (prioritize sequential pieces)  
- **Disk Cache**: Stores downloaded chunks for re-use  
- **HTTP Server**: Serves media with full range requests  
- **Media Player**: mpv, VLC, or browser connects to stream  

---

## 🔧 Configuration Examples

Limit download rate:
```bash
./quickseed "movie.torrent" -r 1048576   # 1 MB/s
```

Choose specific file in a multi-file torrent:
```bash
./quickseed "series.torrent" -f 2
```

Change port & save directory:
```bash
./quickseed "magnet:?xt=..." -p 9000 -d downloads/
```

Disable DHT for privacy:
```bash
./quickseed "file.torrent" --no-dht
```

---

## 📊 Monitoring Stats
QuickSeed prints real-time stats in terminal:

```
[QuickSeed Stats]
Peers: 45 | Download: 2.5 MB/s | Upload: 450 KB/s
Buffered: 120 MB / 1.4 GB (8.5%)
Streaming on http://localhost:8090
```

---

## ❓ Troubleshooting
- **Video lags / buffering** → Increase peers with `--max-peers`  
- **Player doesn’t start** → Use `--player=mpv` or `--player=vlc`  
- **Wrong file streamed** → Use `-f` to select correct file index  
- **No peers found** → Ensure torrent is healthy; check firewall  
- **Port blocked** → Try another port with `--port`  

---

## 📈 Roadmap
- [ ] Subtitles auto-fetching  
- [ ] Web UI with playback controls  
- [ ] Resume unfinished torrents  
- [ ] Native mobile apps  
- [ ] Encrypted peer communication  

---

## 🤝 Contributing
Pull requests are welcome!  
Steps:
1. Fork the repo  
2. Create a feature branch (`git checkout -b feature/fooBar`)  
3. Commit your changes (`git commit -m 'Add fooBar feature'`)  
4. Push to branch (`git push origin feature/fooBar`)  
5. Create a Pull Request  

---

## 🧑‍💻 Maintainers
- [Piyush Golan](https://github.com/yourusername)

---

## 📜 License
MIT License © 2025 QuickSeed Contributors
