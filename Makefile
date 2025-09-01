APP_NAME=quickseed
SOURCE=cmd/quickseed/main.go
BUILD_FLAGS=-ldflags="-s -w"
DIST_DIR=dist

.PHONY: all clean windows linux darwin freebsd

all: windows linux darwin freebsd

clean:
	rm -rf $(DIST_DIR)

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

# Windows builds
windows: $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME).exe $(SOURCE)
	GOOS=windows GOARCH=386 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-386.exe $(SOURCE)
	GOOS=windows GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-arm64.exe $(SOURCE)

# Linux builds
linux: $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(SOURCE)
	GOOS=linux GOARCH=386 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-386 $(SOURCE)
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(SOURCE)
	GOOS=linux GOARCH=arm go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm $(SOURCE)

# macOS builds
darwin: $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 $(SOURCE)
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(SOURCE)

# FreeBSD builds
freebsd: $(DIST_DIR)
	GOOS=freebsd GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-freebsd-amd64 $(SOURCE)

# Individual platform targets
windows-only:
	$(MAKE) windows

linux-only:
	$(MAKE) linux

darwin-only:
	$(MAKE) darwin