# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## IMPORTANT: Communication Language Guidelines
**All user communication MUST be in Korean**
- Respond to users exclusively in Korean
- Code comments: Use English (following project conventions)
- Error messages and technical documentation: Provide BOTH Korean and English versions
  - Example: "파일을 찾을 수 없습니다 (File not found)"
  - Always include both languages for clarity and accessibility
- Internal memory/notes: Store in English for optimal Claude comprehension

## Project Overview

BMW Driving Center Reservation Alert System - BMW 드라이빙 센터 예약 알림 시스템

This Go application monitors the BMW Driving Center website (https://driving-center.bmw.co.kr) for program availability and sends email notifications when desired programs become available for reservation.

### Main Features / 주요 기능
1. **24/7 Monitoring** - Checks reservation page at configurable intervals (minutes/seconds)
   - 24시간 모니터링 - 설정 가능한 간격(분/초)으로 예약 페이지 확인
2. **Program List Tracking** - Monitor multiple desired programs simultaneously
   - 프로그램 리스트 추적 - 원하는 여러 프로그램 동시 모니터링
3. **Email Notifications** - Instant alerts when programs open
   - 이메일 알림 - 프로그램 오픈 시 즉시 알림
4. **Target URLs**:
   - Reservation Page (예약 페이지): https://driving-center.bmw.co.kr/orders/programs/products/view
   - Program List (프로그램 목록): https://driving-center.bmw.co.kr/useAmount/view

## Development Commands

### Go Module Management
- `go mod init` - Initialize a new module (already done)
- `go mod tidy` - Add missing and remove unused modules
- `go mod download` - Download modules to local cache
- `go mod vendor` - Make vendored copy of dependencies

### Building and Running
- `go build` - Build the project
- `go run .` or `go run main.go` - Run the application
- `go build -o bmw-driving-center-alter` - Build with specific output name

### Testing
- `go test ./...` - Run all tests
- `go test -v ./...` - Run tests with verbose output
- `go test -cover ./...` - Run tests with coverage
- `go test -run TestName` - Run specific test by name

### Code Quality
- `go fmt ./...` - Format all Go files
- `go vet ./...` - Examine code for suspicious constructs
- `golangci-lint run` - Run linter (if golangci-lint is installed)

## Project Structure

Recommended structure for BMW Driving Center Alert:

```
/bmw-driving-center-alter
├── /cmd
│   └── /monitor         # Main monitoring application
├── /internal
│   ├── /scraper        # Web scraping logic
│   ├── /notifier       # Email notification service
│   ├── /config         # Configuration management
│   └── /models         # Data models (Program, Notification, etc.)
├── /configs
│   └── config.yaml     # User configuration (programs to monitor, email settings)
├── /logs               # Application logs
└── go.mod
```

### Key Components / 주요 컴포넌트
- **Scraper**: HTTP client for checking reservation page status
  - 스크래퍼: 예약 페이지 상태 확인용 HTTP 클라이언트
- **Notifier**: Email service (SMTP) for sending alerts
  - 알림 서비스: 알림 전송용 이메일 서비스 (SMTP)
- **Scheduler**: Cron-like scheduler for periodic checks
  - 스케줄러: 주기적 확인을 위한 크론 스타일 스케줄러
- **Config Manager**: YAML/JSON config for program lists and settings
  - 설정 관리자: 프로그램 리스트와 설정을 위한 YAML/JSON 설정

## Development Notes

- The project uses Go modules (go.mod) for dependency management
- Currently using Go version 1.24
- This appears to be a GoLand project based on the .idea directory