package main

import (
	"bmw-driving-center-alter/internal/browser"
	"bmw-driving-center-alter/internal/config"
	"bmw-driving-center-alter/internal/models"
	"bmw-driving-center-alter/internal/notifier"
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type GUI struct {
	app            fyne.App
	window         fyne.Window
	config         *config.Config
	configPath     string
	
	// UI components
	usernameEntry  *widget.Entry
	passwordEntry  *widget.Entry
	intervalEntry  *widget.Entry
	emailFromEntry *widget.Entry
	emailToEntry   *widget.Entry
	smtpHostEntry  *widget.Entry
	smtpPortEntry  *widget.Entry
	smtpUserEntry  *widget.Entry
	smtpPassEntry  *widget.Entry
	
	programCheckboxes      map[string]*widget.Check
	selectedProgramsLabel  *widget.Label
	programs              []models.Program
	statusLabel           *widget.Label
	logOutput             *widget.Entry
	activityLog           *widget.Entry
	headlessCheck         *widget.Check
	
	isMonitoring   binding.Bool
	stopChan       chan bool
	browserClient  *browser.BrowserClient
}

func main() {
	gui := &GUI{
		programs: []models.Program{},
	}
	gui.isMonitoring = binding.NewBool()
	
	// 설정 파일 경로 자동 탐색
	configPath := config.GetConfigPath()
	log.Printf("설정 파일 경로: %s", configPath)
	
	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("설정 파일 로드 실패, 기본 설정 생성: %v", err)
		// 기본 설정 생성
		cfg = createDefaultConfig()
		// 설정 파일 저장
		if err := saveDefaultConfig(configPath, cfg); err != nil {
			log.Printf("기본 설정 파일 저장 실패: %v", err)
		}
	}
	gui.config = cfg
	gui.configPath = configPath
	
	// Create app
	gui.app = app.New()
	gui.app.Settings().SetTheme(&myTheme{})
	gui.window = gui.app.NewWindow("BMW 드라이빙 센터 모니터")
	gui.window.Resize(fyne.NewSize(900, 700))
	
	// Build UI
	content := gui.buildUI()
	gui.window.SetContent(content)
	
	// Load config values to UI
	gui.loadConfigToUI()
	
	// 종료 시 정리
	gui.window.SetOnClosed(func() {
		// 모니터링 중이면 중단
		isMonitoring, _ := gui.isMonitoring.Get()
		if isMonitoring {
			log.Println("종료 시 모니터링 중단...")
			gui.stopMonitoring()
			time.Sleep(2 * time.Second) // 브라우저 종료 대기
		}
	})
	
	gui.window.ShowAndRun()
}

func (g *GUI) buildUI() fyne.CanvasObject {
	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("모니터링", g.buildMonitorTab()),
		container.NewTabItem("설정", g.buildSettingsTab()),
		container.NewTabItem("프로그램 목록", g.buildProgramsTab()),
		container.NewTabItem("로그", g.buildLogTab()),
	)
	
	return tabs
}

func (g *GUI) buildMonitorTab() fyne.CanvasObject {
	// Status display
	g.statusLabel = widget.NewLabel("대기 중 (Idle)")
	g.statusLabel.TextStyle.Bold = true
	
	// Control buttons
	startBtn := widget.NewButton("모니터링 시작", func() {
		g.startMonitoring()
	})
	startBtn.Importance = widget.HighImportance
	
	stopBtn := widget.NewButton("모니터링 중지", func() {
		g.stopMonitoring()
	})
	stopBtn.Importance = widget.DangerImportance
	
	testBtn := widget.NewButton("이메일 테스트", func() {
		g.testEmail()
	})
	
	// Update button states based on monitoring status
	g.isMonitoring.AddListener(binding.NewDataListener(func() {
		monitoring, _ := g.isMonitoring.Get()
		if monitoring {
			startBtn.Disable()
			stopBtn.Enable()
		} else {
			startBtn.Enable()
			stopBtn.Disable()
		}
	}))
	
	// Quick status view
	statusCard := widget.NewCard("상태", "", 
		container.NewVBox(
			g.statusLabel,
			widget.NewSeparator(),
			container.NewGridWithColumns(3,
				startBtn,
				stopBtn,
				testBtn,
			),
		),
	)
	
	// Recent activity log
	g.activityLog = widget.NewMultiLineEntry()
	g.activityLog.SetPlaceHolder("모니터링 활동이 여기에 표시됩니다...")
	// Make it read-only
	g.activityLog.OnChanged = func(s string) {
		// Prevent user editing
	}
	
	activityCard := widget.NewCard("최근 활동", "", 
		container.NewScroll(g.activityLog),
	)
	
	return container.NewBorder(
		statusCard,
		nil,
		nil,
		nil,
		activityCard,
	)
}

func (g *GUI) buildSettingsTab() fyne.CanvasObject {
	// Login settings
	g.usernameEntry = widget.NewEntry()
	g.usernameEntry.SetPlaceHolder("BMW ID")
	
	g.passwordEntry = widget.NewPasswordEntry()
	g.passwordEntry.SetPlaceHolder("비밀번호")
	
	loginCard := widget.NewCard("로그인 정보", "", 
		container.New(layout.NewFormLayout(),
			widget.NewLabel("사용자명:"),
			g.usernameEntry,
			widget.NewLabel("비밀번호:"),
			g.passwordEntry,
		),
	)
	
	// Monitoring settings
	g.intervalEntry = widget.NewEntry()
	g.intervalEntry.SetPlaceHolder("300")
	
	g.headlessCheck = widget.NewCheck("백그라운드 모드 (브라우저 숨김)", nil)
	g.headlessCheck.SetChecked(true) // Default to headless
	
	monitorCard := widget.NewCard("모니터링 설정", "",
		container.New(layout.NewFormLayout(),
			widget.NewLabel("확인 간격(초):"),
			g.intervalEntry,
			widget.NewLabel("브라우저 모드:"),
			g.headlessCheck,
		),
	)
	
	// Email settings
	g.emailFromEntry = widget.NewEntry()
	g.emailFromEntry.SetPlaceHolder("sender@gmail.com")
	
	g.emailToEntry = widget.NewEntry()
	g.emailToEntry.SetPlaceHolder("recipient@example.com")
	
	g.smtpHostEntry = widget.NewEntry()
	g.smtpHostEntry.SetPlaceHolder("smtp.gmail.com")
	
	g.smtpPortEntry = widget.NewEntry()
	g.smtpPortEntry.SetPlaceHolder("587")
	
	g.smtpUserEntry = widget.NewEntry()
	g.smtpUserEntry.SetPlaceHolder("your-email@gmail.com")
	
	g.smtpPassEntry = widget.NewPasswordEntry()
	g.smtpPassEntry.SetPlaceHolder("App Password")
	
	emailCard := widget.NewCard("이메일 설정", "",
		container.New(layout.NewFormLayout(),
			widget.NewLabel("보내는 사람:"),
			g.emailFromEntry,
			widget.NewLabel("받는 사람:"),
			g.emailToEntry,
			widget.NewLabel("SMTP 서버:"),
			g.smtpHostEntry,
			widget.NewLabel("SMTP 포트:"),
			g.smtpPortEntry,
			widget.NewLabel("SMTP 사용자:"),
			g.smtpUserEntry,
			widget.NewLabel("SMTP 비밀번호:"),
			g.smtpPassEntry,
		),
	)
	
	// Save button
	saveBtn := widget.NewButton("설정 저장", func() {
		g.saveConfig()
	})
	saveBtn.Importance = widget.HighImportance
	
	return container.NewVBox(
		loginCard,
		monitorCard,
		emailCard,
		container.NewCenter(saveBtn),
	)
}

func (g *GUI) buildProgramsTab() fyne.CanvasObject {
	// Create scrollable container for checkboxes
	checkboxContainer := container.NewVBox()
	
	// Map to store checkbox references
	programCheckboxes := make(map[string]*widget.Check)
	
	// Create program selection by category
	for _, category := range models.AllPrograms {
		// Add category label
		categoryLabel := widget.NewLabelWithStyle(category.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		checkboxContainer.Add(categoryLabel)
		
		// Add checkboxes for each program in category
		for _, programName := range category.Programs {
			// Get Korean name if available
			displayName := programName
			if koreanName, exists := models.ProgramNameMap[programName]; exists {
				displayName = fmt.Sprintf("%s (%s)", koreanName, programName)
			}
			
			// Create checkbox for this program
			checkbox := widget.NewCheck(displayName, func(checked bool) {
				g.updateSelectedPrograms()
				// 선택 변경 시 자동 저장
				g.saveConfig()
			})
			
			// Check if this program is already selected from config
			for _, selectedProg := range g.config.Programs {
				if selectedProg.Name == programName {
					checkbox.Checked = true
					break
				}
			}
			
			programCheckboxes[programName] = checkbox
			checkboxContainer.Add(checkbox)
		}
		
		// Add separator between categories
		checkboxContainer.Add(widget.NewSeparator())
	}
	
	// Store checkboxes reference for later use
	g.programCheckboxes = programCheckboxes
	
	// Selected programs summary
	g.selectedProgramsLabel = widget.NewLabel("선택된 프로그램: 0개")
	updateSelectedCount := func() {
		count := 0
		for _, cb := range programCheckboxes {
			if cb.Checked {
				count++
			}
		}
		g.selectedProgramsLabel.SetText(fmt.Sprintf("선택된 프로그램: %d개", count))
	}
	updateSelectedCount()
	
	// Buttons for select all / deselect all
	selectAllBtn := widget.NewButton("모두 선택", func() {
		for _, cb := range programCheckboxes {
			cb.SetChecked(true)
		}
		updateSelectedCount()
		g.updateSelectedPrograms()
	})
	
	deselectAllBtn := widget.NewButton("모두 해제", func() {
		for _, cb := range programCheckboxes {
			cb.SetChecked(false)
		}
		updateSelectedCount()
		g.updateSelectedPrograms()
	})
	
	controlButtons := container.NewHBox(
		selectAllBtn,
		deselectAllBtn,
		layout.NewSpacer(),
		g.selectedProgramsLabel,
	)
	
	// Scrollable list of checkboxes
	scrollablePrograms := container.NewScroll(checkboxContainer)
	scrollablePrograms.SetMinSize(fyne.NewSize(600, 400))
	
	return container.NewBorder(
		widget.NewCard("프로그램 선택", "모니터링할 프로그램을 선택하세요", controlButtons),
		nil,
		nil,
		nil,
		scrollablePrograms,
	)
}

func (g *GUI) buildLogTab() fyne.CanvasObject {
	g.logOutput = widget.NewMultiLineEntry()
	g.logOutput.SetPlaceHolder("로그가 여기에 표시됩니다...")
	// Make read-only without disabling (to keep text color visible)
	g.logOutput.OnChanged = func(s string) {
		// Allow the change but prevent user editing
		if s != g.logOutput.Text {
			g.logOutput.OnChanged = nil
			g.logOutput.SetText(s)
			g.logOutput.OnChanged = func(string) {}
		}
	}
	
	clearBtn := widget.NewButton("로그 지우기", func() {
		g.logOutput.SetText("")
	})
	
	return container.NewBorder(
		nil,
		clearBtn,
		nil,
		nil,
		container.NewScroll(g.logOutput),
	)
}

func (g *GUI) loadConfigToUI() {
	if g.config == nil {
		return
	}
	
	// Load auth settings
	g.usernameEntry.SetText(g.config.Auth.Username)
	g.passwordEntry.SetText(g.config.Auth.Password)
	
	// Load monitor settings
	g.intervalEntry.SetText(fmt.Sprintf("%d", g.config.Monitor.Interval))
	
	// Load email settings
	g.emailFromEntry.SetText(g.config.Email.From)
	if len(g.config.Email.To) > 0 {
		g.emailToEntry.SetText(g.config.Email.To[0])
	}
	g.smtpHostEntry.SetText(g.config.Email.SMTP.Host)
	g.smtpPortEntry.SetText(fmt.Sprintf("%d", g.config.Email.SMTP.Port))
	g.smtpUserEntry.SetText(g.config.Email.SMTP.Username)
	g.smtpPassEntry.SetText(g.config.Email.SMTP.Password)
	
	// Load programs - 설정에서 불러온 프로그램 목록 설정
	g.programs = g.config.Programs
	
	// 프로그램 체크박스가 생성된 후 선택 상태 업데이트
	if g.programCheckboxes != nil {
		for programName, checkbox := range g.programCheckboxes {
			checkbox.Checked = false // 먼저 모두 해제
			for _, selectedProg := range g.config.Programs {
				if selectedProg.Name == programName {
					checkbox.Checked = true
					break
				}
			}
		}
		g.updateSelectedProgramsLabel()
	}
}

// updateSelectedProgramsLabel updates the selected programs count label
func (g *GUI) updateSelectedProgramsLabel() {
	if g.selectedProgramsLabel != nil {
		g.selectedProgramsLabel.SetText(fmt.Sprintf("선택된 프로그램: %d개", len(g.programs)))
	}
}

func (g *GUI) updateSelectedPrograms() {
	g.programs = []models.Program{}
	
	for programName, checkbox := range g.programCheckboxes {
		if checkbox.Checked {
			// Create program with both English and Korean keywords
			keywords := []string{programName}
			if koreanName, exists := models.ProgramNameMap[programName]; exists {
				keywords = append(keywords, koreanName)
			}
			
			g.programs = append(g.programs, models.Program{
				Name:     programName,
				Keywords: keywords,
			})
		}
	}
	
	// Update label
	if g.selectedProgramsLabel != nil {
		g.selectedProgramsLabel.SetText(fmt.Sprintf("선택된 프로그램: %d개", len(g.programs)))
	}
}

func (g *GUI) saveConfig() {
	// Update config from UI
	g.config.Auth.Username = g.usernameEntry.Text
	g.config.Auth.Password = g.passwordEntry.Text
	
	var interval int
	_, err := fmt.Sscanf(g.intervalEntry.Text, "%d", &interval)
	if err == nil && interval > 0 {
		g.config.Monitor.Interval = interval
	}
	
	g.config.Email.From = g.emailFromEntry.Text
	g.config.Email.To = []string{g.emailToEntry.Text}
	g.config.Email.SMTP.Host = g.smtpHostEntry.Text
	
	var port int
	_, err = fmt.Sscanf(g.smtpPortEntry.Text, "%d", &port)
	if err == nil {
		g.config.Email.SMTP.Port = port
	}
	
	g.config.Email.SMTP.Username = g.smtpUserEntry.Text
	g.config.Email.SMTP.Password = g.smtpPassEntry.Text
	
	g.config.Programs = g.programs
	
	// Save using config package
	err = config.Save(g.configPath, g.config)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}
	
	dialog.ShowInformation("성공", "설정이 저장되었습니다.", g.window)
	g.addLog("설정 저장 완료")
}

func (g *GUI) startMonitoring() {
	isMonitoring, _ := g.isMonitoring.Get()
	if isMonitoring {
		g.addLog("이미 모니터링 중입니다")
		return
	}
	
	// Save config first
	g.saveConfig()
	
	g.isMonitoring.Set(true)
	g.statusLabel.SetText("모니터링 중... (Monitoring)")
	g.addLog("모니터링 시작")
	
	// 중단 채널 생성
	g.stopChan = make(chan bool, 1)
	
	// Start monitoring in goroutine
	go g.runMonitoring()
}

func (g *GUI) stopMonitoring() {
	g.addLog("🛑 모니터링 중지 요청...")
	g.isMonitoring.Set(false)
	
	// 중단 신호 전송
	if g.stopChan != nil {
		select {
		case g.stopChan <- true:
			g.addLog("✅ 중단 신호 전송")
		default:
			// 이미 중단 신호가 있음
		}
	}
	
	// 브라우저 강제 종료
	if g.browserClient != nil {
		g.addLog("🌐 브라우저 강제 종료 중...")
		g.browserClient.Close()
		g.browserClient = nil
		g.addLog("✅ 브라우저 종료 완료")
	}
	
	g.statusLabel.SetText("중지됨 (Stopped)")
	g.addLog("⏹️ 모니터링 완전 중지")
}

func (g *GUI) runMonitoring() {
	// 모든 UI 업데이트를 addLog를 통해 수행
	defer func() {
		if r := recover(); r != nil {
			g.addLog(fmt.Sprintf("❌ 모니터링 오류: %v", r))
			g.stopMonitoring()
		}
	}()
	
	g.addLog("===== 모니터링 시작 =====")
	g.addLog(fmt.Sprintf("⚙️ 설정: 간격 %d초, 프로그램 %d개 선택", g.config.Monitor.Interval, len(g.programs)))
	
	// Initialize browser client
	g.addLog("🌐 Playwright 브라우저 클라이언트 초기화 중...")
	var err error
	g.browserClient, err = browser.NewBrowserClient()
	if err != nil {
		g.addLog(fmt.Sprintf("❌ 브라우저 초기화 실패: %v", err))
		g.stopMonitoring()
		return
	}
	defer func() {
		if g.browserClient != nil {
			g.addLog("🔚 브라우저 정리 중...")
			g.browserClient.Close()
			g.browserClient = nil
		}
	}()
	
	// Start browser with user preference
	headless := g.headlessCheck != nil && g.headlessCheck.Checked
	if headless {
		g.addLog("🤖 백그라운드 모드로 Chromium 브라우저 시작...")
	} else {
		g.addLog("👀 일반 모드로 Chromium 브라우저 시작 (창이 표시됩니다)...")
	}
	
	if err := g.browserClient.Start(headless); err != nil {
		g.addLog(fmt.Sprintf("❌ 브라우저 시작 실패: %v", err))
		g.stopMonitoring()
		return
	}
	g.addLog("✅ 브라우저 시작 완료")
	
	// Login
	// 로그인 상태 확인 및 로그인
	g.addLog("🔍 로그인 상태 확인 중...")
	if !g.browserClient.CheckLoginStatus() {
		g.addLog("🔐 BMW 드라이빙 센터 OAuth2 로그인 시작...")
		g.addLog(fmt.Sprintf("   사용자: %s", g.config.Auth.Username))
		
		if err := g.browserClient.Login(g.config.Auth.Username, g.config.Auth.Password); err != nil {
			g.addLog(fmt.Sprintf("❌ 로그인 실패: %v", err))
			g.addLog("   로그인 정보를 확인해주세요")
			g.stopMonitoring()
			return
		}
		g.addLog("✅ 로그인 성공! 세션 활성화됨")
	} else {
		g.addLog("🎉 저장된 세션이 유효합니다")
	}
	
	// Initialize email notifier
	g.addLog("📧 이메일 알림 서비스 초기화...")
	g.addLog(fmt.Sprintf("   SMTP 서버: %s:%d", g.config.Email.SMTP.Host, g.config.Email.SMTP.Port))
	g.addLog(fmt.Sprintf("   수신자: %s", strings.Join(g.config.Email.To, ", ")))
	emailNotifier := notifier.NewEmailNotifier(g.config.Email)
	lastNotified := make(map[string]time.Time)
	
	// Monitoring loop
	g.addLog(fmt.Sprintf("⏰ %d초 간격으로 모니터링 시작...", g.config.Monitor.Interval))
	ticker := time.NewTicker(time.Duration(g.config.Monitor.Interval) * time.Second)
	defer ticker.Stop()
	
	// Initial check
	g.addLog("🔍 첫 번째 예약 확인 시작...")
	g.checkReservations(g.browserClient, emailNotifier, lastNotified)
	
	checkCount := 1
	for {
		select {
		case <-g.stopChan:
			g.addLog("⏹️ 사용자 요청으로 모니터링 중지")
			return
		case <-ticker.C:
			isMonitoring, _ := g.isMonitoring.Get()
			if !isMonitoring {
				g.addLog("⏹️ 모니터링 상태 변경 감지")
				return
			}
			checkCount++
			g.addLog(fmt.Sprintf("🔄 [확인 #%d] 다시 확인 중...", checkCount))
			g.checkReservations(g.browserClient, emailNotifier, lastNotified)
		}
	}
}

func (g *GUI) checkReservations(browser *browser.BrowserClient, notifier *notifier.EmailNotifier, lastNotified map[string]time.Time) {
	checkTime := time.Now()
	g.addLog(fmt.Sprintf("📍 [%s] 예약 페이지 접속 중...", checkTime.Format("15:04:05")))
	g.addLog(fmt.Sprintf("   URL: %s", g.config.Monitor.ReservationURL))
	
	// Get program names to check
	var programNames []string
	for _, program := range g.programs {
		programNames = append(programNames, program.Name)
	}
	
	if len(programNames) == 0 {
		g.addLog("⚠️ 확인할 프로그램이 선택되지 않았습니다")
		g.addLog("   프로그램 목록 탭에서 원하는 프로그램을 선택해주세요")
		return
	}
	
	g.addLog(fmt.Sprintf("🔎 %d개 프로그램 확인 중...", len(programNames)))
	
	// Check reservation page
	availability, err := browser.CheckReservationPage(programNames)
	if err != nil {
		g.addLog(fmt.Sprintf("❌ 예약 페이지 확인 실패: %v", err))
		g.addLog("   네트워크 연결 또는 사이트 상태를 확인해주세요")
		return
	}
	
	// Check for newly opened programs
	var openPrograms []models.Program
	var newlyOpened []string
	availableCount := 0
	unavailableCount := 0
	
	g.addLog("📋 프로그램 상태:")
	for programName, isAvailable := range availability {
		koreanName := ""
		if kName, exists := models.ProgramNameMap[programName]; exists {
			koreanName = fmt.Sprintf(" (%s)", kName)
		}
		
		if isAvailable {
			availableCount++
			g.addLog(fmt.Sprintf("   ✅ %s%s - 예약 가능!", programName, koreanName))
			
			// Check if we haven't notified recently
			lastTime, exists := lastNotified[programName]
			if !exists || time.Since(lastTime) > time.Hour {
				for _, program := range g.programs {
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
			g.addLog(fmt.Sprintf("   ⭕ %s%s - 예약 불가", programName, koreanName))
		}
	}
	
	g.addLog(fmt.Sprintf("📊 결과: 가능 %d개 / 불가 %d개", availableCount, unavailableCount))
	
	if len(newlyOpened) > 0 {
		g.addLog("━━━━━━━━━━━━━━━━━━━━━━")
		g.addLog("🎉🎉 예약 가능한 프로그램 발견! 🎉🎉")
		for _, name := range newlyOpened {
			if kName, exists := models.ProgramNameMap[name]; exists {
				g.addLog(fmt.Sprintf("   🚗 %s (%s)", name, kName))
			} else {
				g.addLog(fmt.Sprintf("   🚗 %s", name))
			}
		}
		g.addLog("━━━━━━━━━━━━━━━━━━━━━━")
		
		// Create reservation status for notification
		status := &models.ReservationStatus{
			Programs:    openPrograms,
			CheckedAt:   checkTime,
			HasOpenings: true,
		}
		
		// Send notification
		g.addLog("📨 이메일 알림 전송 중...")
		if err := notifier.SendNotification(status); err != nil {
			g.addLog(fmt.Sprintf("❌ 알림 전송 실패: %v", err))
			g.addLog("   이메일 설정을 확인해주세요")
		} else {
			g.addLog("✅ 이메일 알림 전송 완료!")
			g.addLog(fmt.Sprintf("   수신자: %s", strings.Join(g.config.Email.To, ", ")))
		}
	} else if availableCount > 0 {
		g.addLog("ℹ️ 예약 가능한 프로그램이 있지만 이미 알림을 보냈습니다 (1시간 이내)")
	}
	
	// Calculate next check time
	nextCheck := checkTime.Add(time.Duration(g.config.Monitor.Interval) * time.Second)
	g.addLog(fmt.Sprintf("⏱️ 다음 확인: %s", nextCheck.Format("15:04:05")))
	g.addLog("─────────────────────────")
}

func (g *GUI) testEmail() {
	g.addLog("이메일 테스트 전송 중...")
	// TODO: Implement email test
	dialog.ShowInformation("이메일 테스트", "테스트 이메일을 전송했습니다.", g.window)
}

func (g *GUI) addLog(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s\n", timestamp, message)
	cleanMessage := message + "\n"
	
	// Fyne UI 업데이트는 반드시 메인 스레드에서 실행
	if g.app != nil && g.window != nil {
		// RunOnMain을 사용하여 스레드 안전성 보장
		g.window.Canvas().Content().Refresh()
		
		// Add to main log tab
		if g.logOutput != nil {
			current := g.logOutput.Text
			g.logOutput.SetText(current + logMessage)
			// 스크롤을 맨 아래로
			g.logOutput.CursorRow = len(strings.Split(g.logOutput.Text, "\n")) - 1
		}
		
		// Also add to activity log on monitor tab
		if g.activityLog != nil {
			current := g.activityLog.Text
			g.activityLog.SetText(current + cleanMessage)
			// 스크롤을 맨 아래로
			g.activityLog.CursorRow = len(strings.Split(g.activityLog.Text, "\n")) - 1
		}
	}
}

// Helper functions
func splitKeywords(text string) []string {
	var keywords []string
	for _, k := range strings.Split(text, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keywords = append(keywords, k)
		}
	}
	return keywords
}

func joinKeywords(keywords []string) string {
	return strings.Join(keywords, ", ")
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() *config.Config {
	return &config.Config{
		Auth: config.AuthConfig{
			Username: "",
			Password: "",
		},
		Monitor: config.MonitorConfig{
			Interval:       300, // 5분
			ReservationURL: "https://driving-center.bmw.co.kr/orders/programs/products/view",
			ProgramListURL: "https://driving-center.bmw.co.kr/useAmount/view",
		},
		Email: config.EmailConfig{
			From: "",
			To:   []string{},
			Subject: "BMW 드라이빙 센터 예약 오픈 알림",
			SMTP: config.SMTPConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "",
				Password: "",
			},
		},
		Programs: []models.Program{},
	}
}

// saveDefaultConfig saves the default configuration to file
func saveDefaultConfig(path string, cfg *config.Config) error {
	return config.Save(path, cfg)
}