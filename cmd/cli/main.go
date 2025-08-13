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
	"strings"
	"syscall"
	"time"
)

var (
	configPath  string
	headless    bool
	showPrograms bool
	interval    int
)

func init() {
	flag.StringVar(&configPath, "config", "", "설정 파일 경로 (비어있으면 자동 탐색)")
	flag.BoolVar(&headless, "headless", true, "백귳b77c운드 모드 (브라우저 숨김)")
	flag.BoolVar(&showPrograms, "list-programs", false, "사용 가능한 프로그램 목록 표시")
	flag.IntVar(&interval, "interval", 0, "확인 간격(초) - 0이면 설정 파일 값 사용")
}

func main() {
	flag.Parse()

	// 프로그램 목록 표시 모드
	if showPrograms {
		showAvailablePrograms()
		return
	}

	// 설정 파일 로드
	if configPath == "" {
		configPath = config.GetConfigPath()
	}
	log.Printf("설정 파일: %s", configPath)
	
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("❌ 설정 파일 로드 실패: %v", err)
	}

	// CLI 플래그가 설정되면 config의 값을 덮어쓰기
	if interval > 0 {
		cfg.Monitor.Interval = interval
	}
	// headless 플래그가 false로 설정된 경우 config 덮어쓰기
	// (기본값이 true이므로 false일 때만 사용자가 변경한 것)
	if !headless {
		cfg.Monitor.Headless = false
	} else {
		// 명시적으로 true를 원하거나 기본값 사용 시 config 값 유지
		// config에 값이 없으면 true 사용
		if cfg.Monitor.Headless == false {
			// config에서 false로 설정된 경우 유지
		} else {
			cfg.Monitor.Headless = true
		}
	}

	// 설정 확인
	if cfg.Auth.Username == "" || cfg.Auth.Password == "" {
		log.Fatal("❌ 로그인 정보가 설정되지 않았습니다. config.yaml 파일을 확인해주세요.")
	}

	if len(cfg.Programs) == 0 {
		log.Fatal("❌ 모니터링할 프로그램이 선택되지 않았습니다. config.yaml 파일을 확인해주세요.")
	}

	// 시작 메시지
	fmt.Println("========================================")
	fmt.Println("   BMW 드라이빙 센터 예약 모니터 CLI")
	fmt.Println("========================================")
	fmt.Printf("📧 사용자: %s\n", cfg.Auth.Username)
	fmt.Printf("⏱️  간격: %d초\n", cfg.Monitor.Interval)
	fmt.Printf("🎯 프로그램: %d개 선택\n", len(cfg.Programs))
	fmt.Println("----------------------------------------")
	for _, prog := range cfg.Programs {
		koreanName := prog.Name
		if kName, exists := models.ProgramNameMap[prog.Name]; exists {
			koreanName = kName
		}
		fmt.Printf("  • %s\n", koreanName)
	}
	fmt.Println("========================================\n")

	// 시그널 핸들러 설정
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 모니터링 시작
	stopChan := make(chan bool)
	go func() {
		<-sigChan
		fmt.Println("\n\n⏹️  종료 신호 수신... 정리 중...")
		stopChan <- true
	}()

	// 모니터링 실행
	if err := runMonitoring(cfg, stopChan); err != nil {
		log.Fatalf("❌ 모니터링 실행 실패: %v", err)
	}

	fmt.Println("👋 프로그램을 종료합니다.")
}

func runMonitoring(cfg *config.Config, stopChan chan bool) error {
	log.Println("🚀 모니터링 시작...")

	// 브라우저 클라이언트 초기화
	browserClient, err := browser.NewBrowserClientWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("브라우저 초기화 실패: %w", err)
	}
	defer browserClient.Close()

	// 브라우저 시작
	if cfg.Monitor.Headless {
		log.Println("🤖 백그라운드 모드로 브라우저 시작...")
	} else {
		log.Println("👀 일반 모드로 브라우저 시작 (창이 표시됩니다)...")
	}

	if err := browserClient.Start(cfg.Monitor.Headless); err != nil {
		return fmt.Errorf("브라우저 시작 실패: %w", err)
	}

	// 로그인 상태 확인 및 로그인
	log.Println("🔍 로그인 상태 확인 중...")
	if !browserClient.CheckLoginStatus() {
		log.Println("🔐 BMW 드라이빙 센터 로그인 시작...")
		if err := browserClient.Login(cfg.Auth.Username, cfg.Auth.Password); err != nil {
			return fmt.Errorf("로그인 실패: %w", err)
		}
		log.Println("✅ 로그인 성공!")
	} else {
		log.Println("🎉 저장된 세션이 유효합니다")
	}

	// 이메일 알림 초기화
	emailNotifier := notifier.NewEmailNotifier(cfg.Email)
	lastNotified := make(map[string]time.Time)

	// 모니터링 루프
	ticker := time.NewTicker(time.Duration(cfg.Monitor.Interval) * time.Second)
	defer ticker.Stop()

	// 첫 번째 확인
	checkReservations(browserClient, emailNotifier, cfg.Programs, lastNotified)

	checkCount := 1
	for {
		select {
		case <-stopChan:
			return nil
		case <-ticker.C:
			checkCount++
			log.Printf("🔄 [확인 #%d] 예약 상태 확인 중...", checkCount)
			checkReservations(browserClient, emailNotifier, cfg.Programs, lastNotified)
		}
	}
}

func checkReservations(browser *browser.BrowserClient, notifier *notifier.EmailNotifier, programs []models.Program, lastNotified map[string]time.Time) {
	checkTime := time.Now()
	log.Printf("📍 [%s] 예약 페이지 확인 중...", checkTime.Format("15:04:05"))

	// 프로그램 이름 추출
	var programNames []string
	for _, program := range programs {
		programNames = append(programNames, program.Name)
	}

	// 예약 페이지 확인 (hCaptcha 감지 포함)
	availability, captchaDetected, err := browser.CheckReservationPageWithCaptchaAlert(programNames)
	if err != nil {
		log.Printf("❌ 예약 페이지 확인 실패: %v", err)
		return
	}
	
	// hCaptcha가 감지되면 이메일 알림 전송
	if captchaDetected {
		log.Println("📨 CAPTCHA 감지 이메일 알림 전송 중...")
		if err := notifier.SendCaptchaAlert(); err != nil {
			log.Printf("❌ CAPTCHA 알림 전송 실패: %v", err)
		} else {
			log.Println("✅ CAPTCHA 알림 이메일 전송 완료!")
		}
	}

	// 결과 확인
	var openPrograms []models.Program
	var newlyOpened []string
	availableCount := 0
	unavailableCount := 0

	fmt.Println("\n📋 프로그램 상태:")
	for programName, isAvailable := range availability {
		koreanName := ""
		if kName, exists := models.ProgramNameMap[programName]; exists {
			koreanName = fmt.Sprintf(" (%s)", kName)
		}

		if isAvailable {
			availableCount++
			fmt.Printf("   ✅ %s%s - 예약 가능!\n", programName, koreanName)

			// 최근 알림 확인
			lastTime, exists := lastNotified[programName]
			if !exists || time.Since(lastTime) > time.Hour {
				for _, program := range programs {
					if program.Name == programName {
						openPrograms = append(openPrograms, program)
						newlyOpened = append(newlyOpened, programName)
						lastNotified[programName] = time.Now()
						break
					}
				}
			}
		} else {
			unavailableCount++
			fmt.Printf("   ⭕ %s%s - 예약 불가\n", programName, koreanName)
		}
	}

	fmt.Printf("\n📊 결과: 가능 %d개 / 불가 %d개\n", availableCount, unavailableCount)

	// 알림 전송
	if len(newlyOpened) > 0 {
		fmt.Println("\n🎉🎉 예약 가능한 프로그램 발견! 🎉🎉")
		for _, name := range newlyOpened {
			if kName, exists := models.ProgramNameMap[name]; exists {
				fmt.Printf("   🚗 %s (%s)\n", name, kName)
			} else {
				fmt.Printf("   🚗 %s\n", name)
			}
		}

		// 이메일 알림
		status := &models.ReservationStatus{
			Programs:    openPrograms,
			CheckedAt:   checkTime,
			HasOpenings: true,
		}

		fmt.Println("📨 이메일 알림 전송 중...")
		if err := notifier.SendNotification(status); err != nil {
			log.Printf("❌ 알림 전송 실패: %v", err)
		} else {
			fmt.Println("✅ 이메일 알림 전송 완료!")
		}
	}

	// 다음 확인 시간
	nextCheck := checkTime.Add(time.Duration(60) * time.Second)
	fmt.Printf("\n⏱️  다음 확인: %s\n", nextCheck.Format("15:04:05"))
	fmt.Println(strings.Repeat("-", 40))
}

func showAvailablePrograms() {
	fmt.Println("\n=== 사용 가능한 프로그램 목록 ===\n")
	
	for _, category := range models.AllPrograms {
		fmt.Printf("【%s】\n", category.Name)
		for _, program := range category.Programs {
			koreanName := ""
			if kName, exists := models.ProgramNameMap[program]; exists {
				koreanName = fmt.Sprintf(" (%s)", kName)
			}
			fmt.Printf("  • %s%s\n", program, koreanName)
		}
		fmt.Println()
	}
	
	fmt.Println("위 프로그램명을 config.yaml 파일의 programs 섹션에 추가하세요.")
	fmt.Println("예시:")
	fmt.Println("programs:")
	fmt.Println("  - name: M Core")
	fmt.Println("    keywords:")
	fmt.Println("      - M Core")
	fmt.Println("      - M 코어")
}