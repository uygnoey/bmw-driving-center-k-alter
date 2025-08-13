# BMW 드라이빙 센터 모니터 Makefile

# 변수 정의
BUILD_DIR := build
CLI_BINARY := bmw-monitor-cli
GUI_BINARY := bmw-monitor-gui
VERSION := 1.0.0
BUILD_TIME := $(shell date '+%Y-%m-%d_%H:%M:%S')

# Go 파라미터
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# 빌드 플래그
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# OS별 설정
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
	OS := darwin
	ifeq ($(UNAME_M),arm64)
		ARCH := arm64
	else
		ARCH := amd64
	endif
	GUI_EXT :=
else ifeq ($(UNAME_S),Linux)
	OS := linux
	ARCH := amd64
	GUI_EXT :=
else
	OS := windows
	ARCH := amd64
	GUI_EXT := .exe
endif

# 기본 타겟
.PHONY: all
all: clean deps build

# 의존성 설치
.PHONY: deps
deps:
	@echo "📥 의존성 다운로드..."
	$(GOMOD) download
	$(GOMOD) tidy

# 빌드 디렉토리 생성
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# CLI 빌드
.PHONY: cli
cli: $(BUILD_DIR)
	@echo "🔨 CLI 버전 빌드 중..."
	GOOS=$(OS) GOARCH=$(ARCH) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)$(GUI_EXT) cmd/cli/main.go
	@echo "✅ CLI 빌드 완료: $(BUILD_DIR)/$(CLI_BINARY)$(GUI_EXT)"

# GUI 빌드
.PHONY: gui
gui: $(BUILD_DIR)
	@echo "🔨 GUI 버전 빌드 중..."
	GOOS=$(OS) GOARCH=$(ARCH) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(GUI_BINARY)$(GUI_EXT) cmd/gui/*.go
	@echo "✅ GUI 빌드 완료: $(BUILD_DIR)/$(GUI_BINARY)$(GUI_EXT)"

# 전체 빌드
.PHONY: build
build: cli gui copy-configs
	@echo "🎉 빌드 완료!"

# 설정 파일 복사
.PHONY: copy-configs
copy-configs:
	@echo "📄 설정 파일 복사 중..."
	@cp -r configs $(BUILD_DIR)/
	@echo "✅ 설정 파일 복사 완료"

# 모든 플랫폼용 빌드
.PHONY: build-all
build-all: clean deps
	@echo "🌍 모든 플랫폼용 빌드 시작..."
	@mkdir -p $(BUILD_DIR)
	
	# macOS ARM64 (M1/M2)
	@echo "🍎 macOS ARM64 빌드..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-arm64 cmd/cli/main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(GUI_BINARY)-darwin-arm64 cmd/gui/*.go
	
	# macOS AMD64 (Intel)
	@echo "🍎 macOS AMD64 빌드..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-amd64 cmd/cli/main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(GUI_BINARY)-darwin-amd64 cmd/gui/*.go
	
	# Linux AMD64
	@echo "🐧 Linux AMD64 빌드..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-linux-amd64 cmd/cli/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(GUI_BINARY)-linux-amd64 cmd/gui/*.go
	
	# Windows AMD64
	@echo "🪟 Windows AMD64 빌드..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-windows-amd64.exe cmd/cli/main.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(GUI_BINARY)-windows-amd64.exe cmd/gui/*.go
	
	@cp -r configs $(BUILD_DIR)/
	@echo "✅ 모든 플랫폼 빌드 완료!"

# 테스트
.PHONY: test
test:
	@echo "🧪 테스트 실행..."
	$(GOTEST) -v ./...

# 정리
.PHONY: clean
clean:
	@echo "🧹 빌드 파일 정리 중..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "✅ 정리 완료"

# 실행 - CLI
.PHONY: run-cli
run-cli:
	@echo "🚀 CLI 실행..."
	$(GOCMD) run cmd/cli/main.go

# 실행 - GUI
.PHONY: run-gui
run-gui:
	@echo "🚀 GUI 실행..."
	$(GOCMD) run cmd/gui/*.go

# Playwright 브라우저 설치
.PHONY: install-browser
install-browser:
	@echo "🌐 Playwright 브라우저 설치 중..."
	$(GOCMD) run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
	@echo "✅ 브라우저 설치 완료"

# 도움말
.PHONY: help
help:
	@echo "BMW 드라이빙 센터 모니터 - Makefile 사용법"
	@echo ""
	@echo "사용 가능한 명령:"
	@echo "  make all          - 의존성 설치 및 빌드"
	@echo "  make build        - CLI와 GUI 빌드"
	@echo "  make cli          - CLI만 빌드"
	@echo "  make gui          - GUI만 빌드"
	@echo "  make build-all    - 모든 플랫폼용 빌드"
	@echo "  make test         - 테스트 실행"
	@echo "  make clean        - 빌드 파일 정리"
	@echo "  make run-cli      - CLI 실행"
	@echo "  make run-gui      - GUI 실행"
	@echo "  make install-browser - Playwright 브라우저 설치"
	@echo "  make help         - 이 도움말 표시"
	@echo ""
	@echo "현재 시스템: $(OS) $(ARCH)"