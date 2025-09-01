@echo off
setlocal enabledelayedexpansion

set APP_NAME=quickseed
set SOURCE=cmd/quickseed/main.go
set DIST_DIR=dist

:: Get version from git
for /f "tokens=*" %%i in ('git describe --tags --always 2^>nul') do set VERSION=%%i
if "!VERSION!"=="" set VERSION=dev

set BUILD_FLAGS=-ldflags="-s -w -X main.version=!VERSION!"

:: Handle command line arguments
if "%1"=="clean" goto :clean
if "%1"=="windows" goto :windows
if "%1"=="linux" goto :linux  
if "%1"=="darwin" goto :darwin
if "%1"=="all" goto :all
if "%1"=="" goto :all

echo Unknown target: %1
echo Available targets: all, windows, linux, darwin, clean
exit /b 1

:clean
echo Cleaning previous builds...
if exist %DIST_DIR% rmdir /s /q %DIST_DIR%
echo Cleaned!
exit /b 0

:all
echo Building %APP_NAME% (version: !VERSION!)
if not exist %DIST_DIR% mkdir %DIST_DIR%

echo.
echo Building Windows binaries...
call :build_windows
echo.
echo Building Linux binaries...  
call :build_linux
echo.
echo Building macOS binaries...
call :build_darwin
echo.
echo Building FreeBSD binaries...
call :build_freebsd

echo.
echo Build complete!
dir %DIST_DIR%
exit /b 0

:windows
echo Building Windows binaries...
if not exist %DIST_DIR% mkdir %DIST_DIR%
call :build_windows
echo Build complete!
exit /b 0

:linux
echo Building Linux binaries...
if not exist %DIST_DIR% mkdir %DIST_DIR%
call :build_linux
echo Build complete!
exit /b 0

:darwin
echo Building macOS binaries...
if not exist %DIST_DIR% mkdir %DIST_DIR%
call :build_darwin  
echo Build complete!
exit /b 0

:build_windows
echo Building for windows/amd64...
set GOOS=windows
set GOARCH=amd64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%.exe %SOURCE%

echo Building for windows/386...
set GOOS=windows
set GOARCH=386
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-386.exe %SOURCE%

echo Building for windows/arm64...
set GOOS=windows
set GOARCH=arm64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-arm64.exe %SOURCE%
exit /b 0

:build_linux
echo Building for linux/amd64...
set GOOS=linux
set GOARCH=amd64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-linux-amd64 %SOURCE%

echo Building for linux/386...
set GOOS=linux
set GOARCH=386
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-linux-386 %SOURCE%

echo Building for linux/arm64...
set GOOS=linux
set GOARCH=arm64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-linux-arm64 %SOURCE%

echo Building for linux/arm...
set GOOS=linux
set GOARCH=arm
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-linux-arm %SOURCE%
exit /b 0

:build_darwin
echo Building for darwin/amd64...
set GOOS=darwin
set GOARCH=amd64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-darwin-amd64 %SOURCE%

echo Building for darwin/arm64...
set GOOS=darwin
set GOARCH=arm64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-darwin-arm64 %SOURCE%
exit /b 0

:build_freebsd
echo Building for freebsd/amd64...
set GOOS=freebsd
set GOARCH=amd64
go build %BUILD_FLAGS% -o %DIST_DIR%/%APP_NAME%-freebsd-amd64 %SOURCE%
exit /b 0