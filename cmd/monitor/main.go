package main

import (
	"bmw-driving-center-alter/internal/auth"
	"bmw-driving-center-alter/internal/config"
	"bmw-driving-center-alter/internal/notifier"
	"bmw-driving-center-alter/internal/scraper"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", filepath.Join("configs", "config.yaml"), "설정 파일 경로 (Config file path)")
	testEmail := flag.Bool("test-email", false, "이메일 설정 테스트 (Test email configuration)")
	showPrograms := flag.Bool("list-programs", false, "프로그램 목록 확인 (List available programs)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("설정 파일 로드 실패 (Failed to load config): %v", err)
	}

	log.Println("BMW 드라이빙 센터 예약 모니터링 시작 (Starting BMW Driving Center Reservation Monitor)")
	log.Printf("확인 간격: %d초 (Check interval: %d seconds)", cfg.Monitor.Interval, cfg.Monitor.Interval)

	// Initialize components
	authClient, err := auth.NewAuthClient(auth.LoginCredentials{
		Username: cfg.Auth.Username,
		Password: cfg.Auth.Password,
	})
	if err != nil {
		log.Fatalf("인증 클라이언트 초기화 실패 (Failed to initialize auth client): %v", err)
	}

	webScraper := scraper.New(cfg.Monitor.ReservationURL, cfg.Monitor.ProgramListURL)
	emailNotifier := notifier.NewEmailNotifier(cfg.Email)

	// Test email if requested
	if *testEmail {
		log.Println("이메일 테스트 중... (Testing email...)")
		if err := emailNotifier.TestConnection(); err != nil {
			log.Printf("이메일 테스트 실패 (Email test failed): %v", err)
		} else {
			log.Println("이메일 테스트 성공! (Email test successful!)")
		}
		return
	}

	// List programs if requested
	if *showPrograms {
		log.Println("프로그램 목록 가져오는 중... (Fetching program list...)")
		
		// Login first
		if err := authClient.Login(); err != nil {
			log.Printf("로그인 실패 (Login failed): %v", err)
			return
		}
		
		programs, err := webScraper.FetchProgramList()
		if err != nil {
			log.Printf("프로그램 목록 가져오기 실패 (Failed to fetch programs): %v", err)
		} else {
			fmt.Println("\n사용 가능한 프로그램 (Available Programs):")
			for _, prog := range programs {
				fmt.Printf("  - %s\n", prog)
			}
		}
		return
	}

	// Start monitoring
	monitor := &Monitor{
		config:        cfg,
		authClient:    authClient,
		scraper:       webScraper,
		notifier:      emailNotifier,
		lastNotified:  make(map[string]time.Time),
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
			log.Println("모니터링 종료 (Stopping monitor)")
			return
		}
	}
}

type Monitor struct {
	config       *config.Config
	authClient   *auth.AuthClient
	scraper      *scraper.Scraper
	notifier     *notifier.EmailNotifier
	lastNotified map[string]time.Time
}

func (m *Monitor) check() {
	log.Println("예약 페이지 확인 중... (Checking reservation page...)")
	
	// Login if needed
	if !m.authClient.IsLoggedIn() {
		if err := m.authClient.Login(); err != nil {
			log.Printf("로그인 실패 (Login failed): %v", err)
			return
		}
		log.Println("로그인 성공 (Login successful)")
	}

	// Check reservation status
	status, err := m.scraper.CheckReservationStatus(m.config.Programs)
	if err != nil {
		log.Printf("예약 상태 확인 실패 (Failed to check reservation status): %v", err)
		return
	}

	// Check for newly opened programs
	var newlyOpened []string
	for _, program := range status.Programs {
		if program.IsOpen {
			// Check if we haven't notified about this program recently (within 1 hour)
			lastTime, exists := m.lastNotified[program.Name]
			if !exists || time.Since(lastTime) > time.Hour {
				newlyOpened = append(newlyOpened, program.Name)
				m.lastNotified[program.Name] = time.Now()
			}
		}
	}

	if len(newlyOpened) > 0 {
		log.Printf("🎉 예약 가능한 프로그램 발견! (Found available programs!): %v", newlyOpened)
		
		// Send notification
		if err := m.notifier.SendNotification(status); err != nil {
			log.Printf("알림 전송 실패 (Failed to send notification): %v", err)
		} else {
			log.Println("✅ 이메일 알림 전송 완료 (Email notification sent)")
		}
	} else {
		log.Println("현재 예약 가능한 프로그램 없음 (No programs available)")
	}
}