#!/usr/bin/env python3
"""
QuickSeed-Engine Python Video Player Integration
Integrates directly with your Go QuickSeed-Engine backend
"""

import tkinter as tk
from tkinter import ttk, filedialog, messagebox
import cv2
import numpy as np
import threading
import time
import requests
import json
import os
import sys
import subprocess
import signal
import socket
import tempfile
from PIL import Image, ImageTk
import urllib.parse
import queue
import websocket
import platform

class QuickSeedBackend:
    """Manages the QuickSeed-Engine Go backend"""
    
    def __init__(self, engine_path="./quickseed.exe" if platform.system() == "Windows" else "./quickseed"):
        self.engine_path = engine_path
        self.process = None
        self.port = 8080
        self.host = "localhost"
        self.base_url = f"http://{self.host}:{self.port}"
        
    def find_available_port(self):
        """Find an available port for the backend"""
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.bind(('', 0))
            s.listen(1)
            port = s.getsockname()[1]
        return port
    
    def start_engine(self):
        """Start the QuickSeed-Engine backend"""
        if self.is_running():
            return True
            
        try:
            # Check if binary exists
            if not os.path.exists(self.engine_path):
                print(f"QuickSeed binary not found at: {self.engine_path}")
                return False
                
            # Find available port
            self.port = self.find_available_port()
            self.base_url = f"http://{self.host}:{self.port}"
            
            # Start the Go backend process
            cmd = [self.engine_path, "--port", str(self.port)]
            
            if platform.system() == "Windows":
                self.process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    creationflags=subprocess.CREATE_NEW_PROCESS_GROUP
                )
            else:
                self.process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    preexec_fn=os.setsid
                )
            
            # Wait for the backend to start
            for attempt in range(30):  # 30 second timeout
                try:
                    response = requests.get(f"{self.base_url}/health", timeout=1)
                    if response.status_code == 200:
                        print(f"QuickSeed-Engine started on port {self.port}")
                        return True
                except:
                    time.sleep(1)
                    
            print("Failed to start QuickSeed-Engine (timeout)")
            return False
            
        except Exception as e:
            print(f"Error starting QuickSeed-Engine: {e}")
            return False
    
    def stop_engine(self):
        """Stop the QuickSeed-Engine backend"""
        if self.process:
            try:
                if platform.system() == "Windows":
                    self.process.terminate()
                else:
                    os.killpg(os.getpgid(self.process.pid), signal.SIGTERM)
                    
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                if platform.system() == "Windows":
                    self.process.kill()
                else:
                    os.killpg(os.getpgid(self.process.pid), signal.SIGKILL)
            finally:
                self.process = None
                
    def is_running(self):
        """Check if the backend is running"""
        if self.process and self.process.poll() is None:
            return True
        return False
    
    def get_url(self):
        """Get the backend base URL"""
        return self.base_url

class QuickSeedVideoPlayer:
    def __init__(self, root):
        self.root = root
        self.root.title("QuickSeed Video Player")
        self.root.geometry("1400x900")
        self.root.configure(bg='#1e1e1e')
        
        # Backend integration
        self.backend = QuickSeedBackend()
        self.current_torrent = None
        self.available_files = []
        self.ws_connection = None
        
        # Video playback
        self.video_cap = None
        self.is_playing = False
        self.is_fullscreen = False
        self.current_frame = 0
        self.total_frames = 0
        self.video_fps = 30
        self.playback_thread = None
        self.stream_url = None
        
        # UI Setup
        self.setup_ui()
        self.setup_key_bindings()
        
        # Start backend
        self.start_backend()
        
    def setup_ui(self):
        """Setup the user interface"""
        # Create menu
        self.create_menu_bar()
        
        # Main container
        main_container = tk.Frame(self.root, bg='#1e1e1e')
        main_container.pack(fill=tk.BOTH, expand=True, padx=8, pady=8)
        
        # Toolbar
        self.create_toolbar(main_container)
        
        # Video area
        self.create_video_area(main_container)
        
        # Controls
        self.create_controls(main_container)
        
        # Status bar
        self.create_status_bar(main_container)
        
    def create_menu_bar(self):
        """Create application menu bar"""
        menubar = tk.Menu(self.root)
        self.root.config(menu=menubar)
        
        # File menu
        file_menu = tk.Menu(menubar, tearoff=0)
        menubar.add_cascade(label="File", menu=file_menu)
        file_menu.add_command(label="Open Torrent File...", command=self.open_torrent_file, accelerator="Ctrl+O")
        file_menu.add_command(label="Add Magnet Link...", command=self.add_magnet_dialog, accelerator="Ctrl+M")
        file_menu.add_separator()
        file_menu.add_command(label="Exit", command=self.on_closing, accelerator="Ctrl+Q")
        
        # View menu
        view_menu = tk.Menu(menubar, tearoff=0)
        menubar.add_cascade(label="View", menu=view_menu)
        view_menu.add_command(label="Fullscreen", command=self.toggle_fullscreen, accelerator="F11")
        view_menu.add_command(label="Always on Top", command=self.toggle_always_on_top)
        
    def create_toolbar(self, parent):
        """Create toolbar with torrent input"""
        toolbar = tk.Frame(parent, bg='#2d2d30', height=50)
        toolbar.pack(fill=tk.X, pady=(0, 8))
        toolbar.pack_propagate(False)
        
        # Input section
        input_frame = tk.Frame(toolbar, bg='#2d2d30')
        input_frame.pack(fill=tk.BOTH, expand=True, padx=10, pady=8)
        
        tk.Label(input_frame, text="Torrent/Magnet URL:", 
                bg='#2d2d30', fg='#ffffff', font=('Segoe UI', 9)).pack(side=tk.LEFT, padx=(0, 8))
        
        self.url_entry = tk.Entry(input_frame, bg='#3c3c3c', fg='#ffffff', 
                                 insertbackground='#ffffff', font=('Consolas', 9), bd=1, relief=tk.SOLID)
        self.url_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(0, 8))
        self.url_entry.bind('<Return>', lambda e: self.add_torrent())
        
        # Buttons
        tk.Button(input_frame, text="Add Torrent", command=self.add_torrent,
                 bg='#0e639c', fg='#ffffff', font=('Segoe UI', 9, 'bold'),
                 relief=tk.FLAT, padx=12).pack(side=tk.LEFT, padx=2)
        
        tk.Button(input_frame, text="Browse...", command=self.open_torrent_file,
                 bg='#404040', fg='#ffffff', font=('Segoe UI', 9),
                 relief=tk.FLAT, padx=12).pack(side=tk.LEFT, padx=2)
        
        # File selector
        self.file_var = tk.StringVar()
        self.file_selector = ttk.Combobox(input_frame, textvariable=self.file_var, 
                                         state="readonly", width=35, font=('Segoe UI', 9))
        self.file_selector.pack(side=tk.RIGHT, padx=(8, 0))
        self.file_selector.bind('<<ComboboxSelected>>', self.on_file_selection)
        
    def create_video_area(self, parent):
        """Create video display area"""
        video_container = tk.Frame(parent, bg='#000000', relief=tk.SUNKEN, bd=2)
        video_container.pack(fill=tk.BOTH, expand=True, pady=(0, 8))
        
        self.video_canvas = tk.Label(video_container, bg='#000000', 
                                    text="QuickSeed Video Player\n\nDrop a torrent file or add a magnet link to start", 
                                    fg='#888888', font=('Segoe UI', 14))
        self.video_canvas.pack(expand=True, fill=tk.BOTH)
        
        # Drag and drop (basic implementation)
        self.video_canvas.bind('<Button-1>', self.on_video_click)
        
    def create_controls(self, parent):
        """Create playback controls"""
        controls_container = tk.Frame(parent, bg='#2d2d30', height=100)
        controls_container.pack(fill=tk.X, pady=(0, 8))
        controls_container.pack_propagate(False)
        
        # Progress bar
        progress_frame = tk.Frame(controls_container, bg='#2d2d30')
        progress_frame.pack(fill=tk.X, padx=10, pady=(8, 0))
        
        self.progress_var = tk.DoubleVar()
        self.progress_scale = ttk.Scale(progress_frame, from_=0, to=100, orient=tk.HORIZONTAL,
                                       variable=self.progress_var, command=self.on_seek)
        self.progress_scale.pack(fill=tk.X)
        
        # Control buttons
        button_frame = tk.Frame(controls_container, bg='#2d2d30')
        button_frame.pack(fill=tk.X, padx=10, pady=8)
        
        # Playback buttons
        button_style = {'bg': '#404040', 'fg': '#ffffff', 'font': ('Segoe UI', 12, 'bold'),
                       'relief': tk.FLAT, 'width': 4, 'state': tk.DISABLED}
        
        self.play_button = tk.Button(button_frame, text="▶", command=self.toggle_playback, **button_style)
        self.play_button.pack(side=tk.LEFT, padx=2)
        
        self.stop_button = tk.Button(button_frame, text="⏹", command=self.stop_playback, **button_style)
        self.stop_button.pack(side=tk.LEFT, padx=2)
        
        # Volume control
        tk.Label(button_frame, text="Volume:", bg='#2d2d30', fg='#ffffff', 
                font=('Segoe UI', 9)).pack(side=tk.LEFT, padx=(20, 5))
        
        self.volume_var = tk.DoubleVar(value=75)
        volume_scale = ttk.Scale(button_frame, from_=0, to=100, orient=tk.HORIZONTAL,
                               variable=self.volume_var, length=120)
        volume_scale.pack(side=tk.LEFT, padx=5)
        
        # Time display
        self.time_display = tk.Label(button_frame, text="00:00 / 00:00", 
                                    bg='#2d2d30', fg='#ffffff', font=('Consolas', 10))
        self.time_display.pack(side=tk.RIGHT, padx=10)
        
        # Download progress
        self.download_progress = tk.Label(button_frame, text="", 
                                         bg='#2d2d30', fg='#00ff00', font=('Segoe UI', 9))
        self.download_progress.pack(side=tk.RIGHT, padx=10)
        
    def create_status_bar(self, parent):
        """Create status bar"""
        status_frame = tk.Frame(parent, bg='#2d2d30', height=25, relief=tk.SUNKEN, bd=1)
        status_frame.pack(fill=tk.X)
        status_frame.pack_propagate(False)
        
        self.status_label = tk.Label(status_frame, text="Starting QuickSeed-Engine...", 
                                    bg='#2d2d30', fg='#ffffff', font=('Segoe UI', 9), anchor=tk.W)
        self.status_label.pack(side=tk.LEFT, padx=5, pady=2)
        
        self.backend_status = tk.Label(status_frame, text="●", 
                                      bg='#2d2d30', fg='#ff0000', font=('Segoe UI', 12))
        self.backend_status.pack(side=tk.RIGHT, padx=5, pady=2)
        
    def setup_key_bindings(self):
        """Setup keyboard shortcuts"""
        self.root.bind('<Control-o>', lambda e: self.open_torrent_file())
        self.root.bind('<Control-m>', lambda e: self.add_magnet_dialog())
        self.root.bind('<Control-q>', lambda e: self.on_closing())
        self.root.bind('<F11>', lambda e: self.toggle_fullscreen())
        self.root.bind('<space>', lambda e: self.toggle_playback())
        self.root.bind('<Escape>', lambda e: self.exit_fullscreen())
        
    def start_backend(self):
        """Start the QuickSeed-Engine backend"""
        def start_thread():
            if self.backend.start_engine():
                self.root.after(0, lambda: self.update_status("QuickSeed-Engine started successfully"))
                self.root.after(0, lambda: self.backend_status.config(fg='#00ff00'))
                self.root.after(0, self.setup_websocket)
            else:
                self.root.after(0, lambda: self.update_status("Failed to start QuickSeed-Engine"))
                self.root.after(0, lambda: self.backend_status.config(fg='#ff0000'))
                
        threading.Thread(target=start_thread, daemon=True).start()
        
    def setup_websocket(self):
        """Setup WebSocket connection for real-time updates"""
        def on_message(ws, message):
            try:
                data = json.loads(message)
                self.root.after(0, lambda: self.handle_websocket_message(data))
            except Exception as e:
                print(f"WebSocket message error: {e}")
                
        def on_error(ws, error):
            print(f"WebSocket error: {error}")
            
        def on_close(ws, close_status_code, close_msg):
            print("WebSocket connection closed")
            
        def on_open(ws):
            print("WebSocket connection established")
            
        try:
            ws_url = f"ws://localhost:{self.backend.port}/ws"
            self.ws_connection = websocket.WebSocketApp(
                ws_url,
                on_open=on_open,
                on_message=on_message,
                on_error=on_error,
                on_close=on_close
            )
            
            ws_thread = threading.Thread(target=self.ws_connection.run_forever, daemon=True)
            ws_thread.start()
            
        except Exception as e:
            print(f"WebSocket setup failed: {e}")
            
    def handle_websocket_message(self, data):
        """Handle WebSocket messages from backend"""
        msg_type = data.get('type')
        
        if msg_type == 'download_progress':
            progress = data.get('progress', 0)
            speed = data.get('speed', 0)
            self.download_progress.config(text=f"↓ {progress:.1f}% ({speed:.1f} KB/s)")
            
        elif msg_type == 'torrent_added':
            self.current_torrent = data.get('torrent_id')
            self.update_status(f"Torrent added: {data.get('name', 'Unknown')}")
            self.refresh_file_list()
            
        elif msg_type == 'files_available':
            self.refresh_file_list()
            
        elif msg_type == 'error':
            self.update_status(f"Error: {data.get('message', 'Unknown error')}")
            
    def add_torrent(self):
        """Add torrent from URL entry"""
        url = self.url_entry.get().strip()
        if not url:
            messagebox.showerror("Error", "Please enter a torrent URL or magnet link")
            return
            
        self.add_torrent_url(url)
        
    def add_torrent_url(self, url):
        """Add torrent via URL/magnet"""
        def add_thread():
            try:
                self.root.after(0, lambda: self.update_status("Adding torrent..."))
                
                if url.startswith("magnet:"):
                    payload = {"magnet_link": url}
                else:
                    payload = {"torrent_url": url}
                    
                response = requests.post(f"{self.backend.get_url()}/api/torrents", 
                                       json=payload, timeout=30)
                
                if response.status_code == 200:
                    result = response.json()
                    self.current_torrent = result.get('torrent_id')
                    self.root.after(0, lambda: self.update_status("Torrent added successfully"))
                    self.root.after(0, self.refresh_file_list)
                else:
                    error = f"Failed to add torrent: {response.text}"
                    self.root.after(0, lambda: self.update_status(error))
                    
            except Exception as e:
                error = f"Error adding torrent: {str(e)}"
                self.root.after(0, lambda: self.update_status(error))
                
        threading.Thread(target=add_thread, daemon=True).start()
        
    def open_torrent_file(self):
        """Open torrent file dialog"""
        file_path = filedialog.askopenfilename(
            title="Select Torrent File",
            filetypes=[("Torrent files", "*.torrent"), ("All files", "*.*")]
        )
        
        if file_path:
            self.upload_torrent_file(file_path)
            
    def upload_torrent_file(self, file_path):
        """Upload torrent file to backend"""
        def upload_thread():
            try:
                self.root.after(0, lambda: self.update_status("Uploading torrent file..."))
                
                with open(file_path, 'rb') as f:
                    files = {'torrent_file': f}
                    response = requests.post(f"{self.backend.get_url()}/api/torrents/upload", 
                                           files=files, timeout=30)
                
                if response.status_code == 200:
                    result = response.json()
                    self.current_torrent = result.get('torrent_id')
                    self.root.after(0, lambda: self.update_status("Torrent file uploaded successfully"))
                    self.root.after(0, self.refresh_file_list)
                else:
                    error = f"Failed to upload torrent: {response.text}"
                    self.root.after(0, lambda: self.update_status(error))
                    
            except Exception as e:
                error = f"Error uploading torrent: {str(e)}"
                self.root.after(0, lambda: self.update_status(error))
                
        threading.Thread(target=upload_thread, daemon=True).start()
        
    def add_magnet_dialog(self):
        """Show magnet link input dialog"""
        dialog = tk.Toplevel(self.root)
        dialog.title("Add Magnet Link")
        dialog.geometry("500x150")
        dialog.configure(bg='#2d2d30')
        dialog.transient(self.root)
        dialog.grab_set()
        
        tk.Label(dialog, text="Enter Magnet Link:", bg='#2d2d30', fg='#ffffff', 
                font=('Segoe UI', 10)).pack(pady=10)
        
        entry = tk.Entry(dialog, width=60, bg='#3c3c3c', fg='#ffffff', 
                        insertbackground='#ffffff', font=('Consolas', 9))
        entry.pack(pady=10, padx=20, fill=tk.X)
        entry.focus()
        
        button_frame = tk.Frame(dialog, bg='#2d2d30')
        button_frame.pack(pady=10)
        
        def on_add():
            magnet = entry.get().strip()
            if magnet:
                self.add_torrent_url(magnet)
                dialog.destroy()
            else:
                messagebox.showerror("Error", "Please enter a magnet link")
                
        tk.Button(button_frame, text="Add", command=on_add, bg='#0e639c', fg='#ffffff',
                 font=('Segoe UI', 9, 'bold'), padx=20).pack(side=tk.LEFT, padx=5)
        
        tk.Button(button_frame, text="Cancel", command=dialog.destroy, bg='#404040', fg='#ffffff',
                 font=('Segoe UI', 9), padx=20).pack(side=tk.LEFT, padx=5)
        
        entry.bind('<Return>', lambda e: on_add())
        
    def refresh_file_list(self):
        """Refresh available files from backend"""
        if not self.current_torrent:
            return
            
        def refresh_thread():
            try:
                response = requests.get(f"{self.backend.get_url()}/api/torrents/{self.current_torrent}/files")
                if response.status_code == 200:
                    files = response.json().get('files', [])
                    
                    # Filter video files
                    video_extensions = ['.mp4', '.avi', '.mkv', '.mov', '.wmv', '.flv', '.webm', '.m4v', '.ts', '.m2ts']
                    video_files = [f for f in files if any(f['name'].lower().endswith(ext) for ext in video_extensions)]
                    
                    self.available_files = video_files
                    file_names = [f['name'] for f in video_files]
                    
                    self.root.after(0, lambda: self.file_selector.config(values=file_names))
                    
                    if video_files:
                        self.root.after(0, lambda: self.file_selector.set(file_names[0]))
                        self.root.after(0, lambda: self.update_status(f"Found {len(video_files)} video files"))
                    else:
                        self.root.after(0, lambda: self.update_status("No video files found"))
                        
            except Exception as e:
                error = f"Error refreshing files: {str(e)}"
                self.root.after(0, lambda: self.update_status(error))
                
        threading.Thread(target=refresh_thread, daemon=True).start()
        
    def on_file_selection(self, event=None):
        """Handle file selection from combobox"""
        selected_file = self.file_var.get()
        if not selected_file or not self.current_torrent:
            return
            
        # Find file info
        file_info = None
        for f in self.available_files:
            if f['name'] == selected_file:
                file_info = f
                break
                
        if file_info:
            # Build stream URL
            encoded_filename = urllib.parse.quote(file_info['name'])
            self.stream_url = f"{self.backend.get_url()}/api/stream/{self.current_torrent}/{encoded_filename}"
            
            self.load_video_stream(selected_file)
            
    def load_video_stream(self, filename):
        """Load video stream for playback"""
        def load_thread():
            try:
                self.root.after(0, lambda: self.update_status(f"Loading video: {filename}"))
                
                # Open video stream with OpenCV
                cap = cv2.VideoCapture(self.stream_url)
                
                if cap.isOpened():
                    self.video_cap = cap
                    self.total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
                    self.video_fps = cap.get(cv2.CAP_PROP_FPS) or 30
                    
                    # Enable controls
                    self.root.after(0, lambda: self.play_button.config(state=tk.NORMAL))
                    self.root.after(0, lambda: self.stop_button.config(state=tk.NORMAL))
                    
                    self.root.after(0, lambda: self.update_status(f"Ready to play: {filename}"))
                else:
                    self.root.after(0, lambda: self.update_status(f"Failed to load video: {filename}"))
                    
            except Exception as e:
                error = f"Error loading video: {str(e)}"
                self.root.after(0, lambda: self.update_status(error))
                
        threading.Thread(target=load_thread, daemon=True).start()
        
    def toggle_playback(self):
        """Toggle play/pause"""
        if not self.video_cap:
            return
            
        if self.is_playing:
            self.pause_video()
        else:
            self.play_video()
            
    def play_video(self):
        """Start video playback"""
        if not self.video_cap:
            return
            
        self.is_playing = True
        self.play_button.config(text="⏸")
        
        if not self.playback_thread or not self.playback_thread.is_alive():
            self.playback_thread = threading.Thread(target=self.playback_loop, daemon=True)
            self.playback_thread.start()
            
    def pause_video(self):
        """Pause video playback"""
        self.is_playing = False
        self.play_button.config(text="▶")
        
    def stop_playback(self):
        """Stop video playback"""
        self.is_playing = False
        self.current_frame = 0
        
        if self.video_cap:
            self.video_cap.set(cv2.CAP_PROP_POS_FRAMES, 0)
            
        self.play_button.config(text="▶")
        self.progress_var.set(0)
        self.update_status("Stopped")
        
    def on_seek(self, value):
        """Handle seek operation"""
        if not self.video_cap or self.total_frames == 0:
            return
            
        progress = float(value)
        target_frame = int((progress / 100) * self.total_frames)
        
        self.video_cap.set(cv2.CAP_PROP_POS_FRAMES, target_frame)
        self.current_frame = target_frame
        
    def playback_loop(self):
        """Main video playback loop"""
        while self.is_playing and self.video_cap and self.video_cap.isOpened():
            ret, frame = self.video_cap.read()
            
            if not ret:
                self.root.after(0, self.pause_video)
                self.root.after(0, lambda: self.update_status("End of video"))
                break
                
            # Resize frame for display
            display_frame = self.resize_frame_for_display(frame)
            
            if display_frame is not None:
                # Convert to RGB and create PhotoImage
                rgb_frame = cv2.cvtColor(display_frame, cv2.COLOR_BGR2RGB)
                img = Image.fromarray(rgb_frame)
                photo = ImageTk.PhotoImage(img)
                
                # Update display
                self.root.after(0, lambda p=photo: self.update_video_display(p))
                
            # Update progress and time
            self.current_frame = int(self.video_cap.get(cv2.CAP_PROP_POS_FRAMES))
            if self.total_frames > 0:
                progress = (self.current_frame / self.total_frames) * 100
                self.root.after(0, lambda p=progress: self.progress_var.set(p))
                
            # Update time display
            current_time = self.current_frame / self.video_fps
            total_time = self.total_frames / self.video_fps
            time_str = f"{self.format_time(current_time)} / {self.format_time(total_time)}"
            self.root.after(0, lambda t=time_str: self.time_display.config(text=t))
            
            # Control playback speed
            time.sleep(1.0 / max(self.video_fps, 1))
            
    def resize_frame_for_display(self, frame):
        """Resize frame to fit display area"""
        if frame is None:
            return None
            
        canvas_width = self.video_canvas.winfo_width()
        canvas_height = self.video_canvas.winfo_height()
        
        if canvas_width <= 1 or canvas_height <= 1:
            return frame
            
        frame_height, frame_width = frame.shape[:2]
        
        # Calculate scaling to fit while maintaining aspect ratio
        scale_w = canvas_width / frame_width
        scale_h = canvas_height / frame_height
        scale = min(scale_w, scale_h)
        
        new_width = int(frame_width * scale)
        new_height = int(frame_height * scale)
        
        return cv2.resize(frame, (new_width, new_height))
        
    def update_video_display(self, photo):
        """Update video display with new frame"""
        self.video_canvas.configure(image=photo, text="")
        self.video_canvas.image = photo  # Keep a reference
        
    def format_time(self, seconds):
        """Format time as MM:SS"""
        minutes = int(seconds // 60)
        seconds = int(seconds % 60)
        return f"{minutes:02d}:{seconds:02d}"
        
    def update_status(self, message):
        """Update status bar message"""
        self.status_label.config(text=message)
        
    def on_video_click(self, event):
        """Handle video area clicks"""
        if self.video_cap:
            self.toggle_playback()
            
    def toggle_fullscreen(self):
        """Toggle fullscreen mode"""
        self.is_fullscreen = not self.is_fullscreen
        self.root.attributes('-fullscreen', self.is_fullscreen)
        
        if self.is_fullscreen:
            # Hide UI elements in fullscreen
            self.root.config(cursor="none")
        else:
            self.root.config(cursor="")
            
    def exit_fullscreen(self):
        """Exit fullscreen mode"""
        if self.is_fullscreen:
            self.is_fullscreen = False
            self.root.attributes('-fullscreen', False)
            self.root.config(cursor="")
            
    def toggle_always_on_top(self):
        """Toggle always on top"""
        current = self.root.attributes('-topmost')
        self.root.attributes('-topmost', not current)
        
    def on_closing(self):
        """Handle application closing"""
        # Stop playback
        self.is_playing = False
        
        # Release video resources
        if self.video_cap:
            self.video_cap.release()
            
        # Close WebSocket
        if self.ws_connection:
            self.ws_connection.close()
            
        # Stop backend
        self.backend.stop_engine()
        
        # Destroy window
        self.root.destroy()

class QuickSeedIntegration:
    """Integration layer between Python and Go QuickSeed-Engine"""
    
    @staticmethod
    def build_go_backend():
        """Build the Go backend if not already built"""
        if not os.path.exists("./quickseed") and not os.path.exists("./quickseed.exe"):
            print("Building QuickSeed-Engine...")
            try:
                # Try to build the Go backend
                result = subprocess.run(["go", "build", "-o", "quickseed"], 
                                      capture_output=True, text=True)
                if result.returncode == 0:
                    print("QuickSeed-Engine built successfully")
                    return True
                else:
                    print(f"Build failed: {result.stderr}")
                    return False
            except FileNotFoundError:
                print("Go not found. Please ensure Go is installed and in PATH")
                return False
        return True
    
    @staticmethod
    def check_dependencies():
        """Check if all Python dependencies are available"""
        required_packages = [
            'cv2', 'PIL', 'requests', 'websocket', 'numpy'
        ]
        
        missing_packages = []
        for package in required_packages:
            try:
                __import__(package)
            except ImportError:
                if package == 'cv2':
                    missing_packages.append('opencv-python')
                elif package == 'PIL':
                    missing_packages.append('pillow')
                else:
                    missing_packages.append(package)
        
        if missing_packages:
            print("Missing required packages:")
            for package in missing_packages:
                print(f"  pip install {package}")
            return False
        
        return True

def main():
    """Main application entry point"""
    print("QuickSeed Video Player - Python Integration")
    print("=" * 50)
    
    # Check dependencies
    if not QuickSeedIntegration.check_dependencies():
        print("\nPlease install missing dependencies and try again.")
        sys.exit(1)
    
    # Try to build Go backend if needed
    if not QuickSeedIntegration.build_go_backend():
        print("\nFailed to build QuickSeed-Engine backend.")
        print("Please build manually with: go build -o quickseed")
        sys.exit(1)
    
    # Create and run application
    try:
        root = tk.Tk()
        
        # Set application icon if available
        try:
            root.iconbitmap("icon.ico")  # Add your icon file
        except:
            pass
            
        app = QuickSeedVideoPlayer(root)
        
        # Handle Ctrl+C gracefully
        def signal_handler(signum, frame):
            app.on_closing()
            
        signal.signal(signal.SIGINT, signal_handler)
        
        # Center window on screen
        root.update_idletasks()
        width = root.winfo_width()
        height = root.winfo_height()
        x = (root.winfo_screenwidth() // 2) - (width // 2)
        y = (root.winfo_screenheight() // 2) - (height // 2)
        root.geometry(f'{width}x{height}+{x}+{y}')
        
        # Set window close protocol
        root.protocol("WM_DELETE_WINDOW", app.on_closing)
        
        print("Starting application...")
        root.mainloop()
        
    except KeyboardInterrupt:
        print("\nApplication interrupted by user")
    except Exception as e:
        print(f"Application error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()