package main

import (
	"bmw-driving-center-alter/internal/browser"
	"bmw-driving-center-alter/internal/config"
	"bmw-driving-center-alter/internal/models"
	"bmw-driving-center-alter/internal/notifier"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", filepath.Join("configs", "config.yaml"), "설정 파일 경로")
	headless := flag.Bool("headless", true, "헤드리스 모드 (백그라운드 실행)")
	testLogin := flag.Bool("test-login", false, "로그인 테스트만 수행")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("설정 파일 로드 실패: %v", err)
	}

	log.Println("🚗 BMW 드라이빙 센터 예약 모니터링 시작 (브라우저 모드)")
	log.Printf("확인 간격: %d초", cfg.Monitor.Interval)

	// Initialize browser client
	browserClient, err := browser.NewBrowserClient()
	if err != nil {
		log.Fatalf("브라우저 클라이언트 초기화 실패: %v", err)
	}
	defer browserClient.Close()

	// Start browser
	log.Printf("브라우저 시작 중... (headless=%v)", *headless)
	if err := browserClient.Start(*headless); err != nil {
		log.Fatalf("브라우저 시작 실패: %v", err)
	}

	// Login
	log.Println("로그인 시도 중...")
	if err := browserClient.Login(cfg.Auth.Username, cfg.Auth.Password); err != nil {
		log.Fatalf("로그인 실패: %v", err)
	}
	log.Println("✅ 로그인 성공!")

	// If test login only, exit here
	if *testLogin {
		log.Println("로그인 테스트 완료")
		return
	}

	// Initialize email notifier
	emailNotifier := notifier.NewEmailNotifier(cfg.Email)

	// Start monitoring
	monitor := &BrowserMonitor{
		config:       cfg,
		browser:      browserClient,
		notifier:     emailNotifier,
		lastNotified: make(map[string]time.Time),
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring loop
	ticker := time.NewTicker(time.Duration(cfg.Monitor.Interval) * time.Second)
	defer ticker.Stop()

	// Check immediately on start
	monitor.check()

	for {
		select {
		case <-ticker.C:
			monitor.check()
		case <-sigChan:
			log.Println("모니터링 종료...")
			return
		}
	}
}

type BrowserMonitor struct {
	config       *config.Config
	browser      *browser.BrowserClient
	notifier     *notifier.EmailNotifier
	lastNotified map[string]time.Time
}

func (m *BrowserMonitor) check() {
	log.Println("예약 페이지 확인 중...")

	// Get program names to check
	var programNames []string
	for _, program := range m.config.Programs {
		programNames = append(programNames, program.Name)
	}

	// Check reservation page
	availability, err := m.browser.CheckReservationPage(programNames)
	if err != nil {
		log.Printf("예약 페이지 확인 실패: %v", err)
		return
	}

	// Check for newly opened programs
	var openPrograms []models.Program
	var newlyOpened []string

	for programName, isAvailable := range availability {
		if isAvailable {
			// Find the program in config
			for _, program := range m.config.Programs {
				if program.Name == programName {
					// Check if we haven't notified recently (within 1 hour)
					lastTime, exists := m.lastNotified[programName]
					if !exists || time.Since(lastTime) > time.Hour {
						openPrograms = append(openPrograms, program)
						newlyOpened = append(newlyOpened, programName)
						m.lastNotified[programName] = time.Now()
					}
					break
				}
			}
		}
	}

	if len(newlyOpened) > 0 {
		log.Printf("🎉 예약 가능한 프로그램 발견: %s", strings.Join(newlyOpened, ", "))

		// Create reservation status for notification
		status := &models.ReservationStatus{
			Programs:    openPrograms,
			CheckedAt:   time.Now(),
			HasOpenings: true,
		}

		// Send notification
		if err := m.notifier.SendNotification(status); err != nil {
			log.Printf("알림 전송 실패: %v", err)
		} else {
			log.Println("✅ 이메일 알림 전송 완료")
		}
	} else {
		// Log status for all programs
		log.Println("프로그램 상태:")
		for name, available := range availability {
			status := "❌ 예약 불가"
			if available {
				status = "✅ 예약 가능"
			}
			fmt.Printf("  %s: %s\n", name, status)
		}
	}
}