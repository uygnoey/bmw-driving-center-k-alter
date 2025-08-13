package browser

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// BrowserClient handles browser-based authentication and scraping
type BrowserClient struct {
	pw           *playwright.Playwright
	browser      playwright.Browser
	context      playwright.BrowserContext
	page         playwright.Page
	baseURL      string
	stateDir     string
	isLoggedIn   bool
}

// NewBrowserClient creates a new browser client
func NewBrowserClient() (*BrowserClient, error) {
	// Don't auto-install browsers, use already installed Chromium
	// Browsers should be installed manually with:
	// go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
	
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("Playwright 시작 실패 (failed to start Playwright): %w", err)
	}

	// 세션 저장 디렉토리 설정
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".bmw-driving-center", "browser-state")
	
	// 디렉토리 생성
	err = os.MkdirAll(stateDir, 0755)
	if err != nil {
		log.Printf("⚠️ 세션 디렉토리 생성 실패: %v", err)
	}

	return &BrowserClient{
		pw:         pw,
		baseURL:    "https://driving-center.bmw.co.kr",
		stateDir:   stateDir,
		isLoggedIn: false,
	}, nil
}

// Start launches the browser
func (b *BrowserClient) Start(headless bool) error {
	browser, err := b.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-setuid-sandbox",
		},
	})
	if err != nil {
		return fmt.Errorf("브라우저 실행 실패 (failed to launch browser): %w", err)
	}
	b.browser = browser

	// 저장된 세션 파일 경로
	stateFile := filepath.Join(b.stateDir, "state.json")
	
	// 저장된 세션이 있는지 확인
	if _, err := os.Stat(stateFile); err == nil {
		log.Println("💾 저장된 세션 발견, 복원 시도...")
		
		// 저장된 세션으로 컨텍스트 생성
		context, err := browser.NewContext(playwright.BrowserNewContextOptions{
			StorageStatePath: playwright.String(stateFile),
			UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			Viewport: &playwright.Size{
				Width:  1280,
				Height: 720,
			},
		})
		if err != nil {
			log.Printf("⚠️ 세션 복원 실패, 새 세션 생성: %v", err)
			// 실패 시 새 컨텍스트 생성
			context, err = b.createNewContext(browser)
			if err != nil {
				return err
			}
		} else {
			log.Println("✅ 세션 복원 성공")
		}
		b.context = context
	} else {
		log.Println("🆕 새 세션 생성...")
		context, err := b.createNewContext(browser)
		if err != nil {
			return err
		}
		b.context = context
	}

	page, err := b.context.NewPage()
	if err != nil {
		return fmt.Errorf("페이지 생성 실패 (failed to create page): %w", err)
	}
	
	// Set longer timeout for page operations
	page.SetDefaultTimeout(60000) // 60 seconds
	page.SetDefaultNavigationTimeout(60000)
	
	b.page = page

	return nil
}

// createNewContext creates a new browser context
func (b *BrowserClient) createNewContext(browser playwright.Browser) (playwright.BrowserContext, error) {
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Viewport: &playwright.Size{
			Width:  1280,
			Height: 720,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("브라우저 컨텍스트 생성 실패 (failed to create context): %w", err)
	}
	return context, nil
}

// SaveSession saves the current browser session
func (b *BrowserClient) SaveSession() error {
	if b.context == nil {
		return fmt.Errorf("컨텍스트가 없음")
	}
	
	stateFile := filepath.Join(b.stateDir, "state.json")
	log.Printf("💾 세션 저장 중: %s", stateFile)
	
	// StorageState 메서드에 파일 경로 전달
	_, err := b.context.StorageState(stateFile)
	if err != nil {
		return fmt.Errorf("세션 저장 실패: %w", err)
	}
	
	log.Println("✅ 세션 저장 성공")
	return nil
}

// CheckLoginStatus checks if already logged in
func (b *BrowserClient) CheckLoginStatus() bool {
	log.Println("🔍 로그인 상태 확인 중...")
	
	// 먼저 메인 페이지로 이동
	log.Printf("1️⃣ BMW 드라이빙 센터 메인 페이지 접속: %s", b.baseURL)
	_, err := b.page.Goto(b.baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		log.Printf("⚠️ 메인 페이지 접속 실패: %v", err)
	}
	
	// 예약 페이지로 이동 시도
	log.Println("2️⃣ 예약 페이지로 이동 시도...")
	_, err = b.page.Goto(b.baseURL + "/orders/programs/products/view", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(15000), // 30초에서 15초로 단축
	})
	if err != nil {
		log.Printf("⚠️ 예약 페이지 이동 경고: %v", err)
	}
	
	// 페이지 안정화를 위한 최소 대기
	time.Sleep(1 * time.Second)
	currentURL := b.page.URL()
	log.Printf("📍 현재 URL: %s", currentURL)
	
	// 로그인 페이지로 리다이렉트되지 않으면 로그인된 상태
	if strings.Contains(currentURL, "driving-center.bmw.co.kr/orders") {
		log.Println("✅ 이미 로그인되어 있음 (세션 유효)")
		b.isLoggedIn = true
		return true
	}
	
	// customer.bmwgroup.com으로 리다이렉트되면 로그인 필요
	if strings.Contains(currentURL, "customer.bmwgroup.com") {
		log.Println("⚠️ 로그인 페이지로 리다이렉트됨 - 로그인 필요")
		b.isLoggedIn = false
		return false
	}
	
	// 기타 페이지인 경우
	log.Printf("⚠️ 예상치 못한 페이지: %s", currentURL)
	b.isLoggedIn = false
	return false
}

// Login performs OAuth2 login to BMW Driving Center  
func (b *BrowserClient) Login(username, password string) error {
	log.Println("===== BMW 드라이빙 센터 로그인 시작 =====")
	
	// 현재 페이지 URL 확인
	currentURL := b.page.URL()
	log.Printf("📍 현재 페이지: %s", currentURL)
	
	// 이미 로그인 페이지에 있는지 확인
	if strings.Contains(currentURL, "customer.bmwgroup.com") {
		log.Println("✅ 이미 BMW 로그인 페이지에 있음")
		// 바로 로그인 진행
	} else {
		// BMW 드라이빙 센터에서 로그인 상태 재확인
		log.Println("🔄 BMW 드라이빙 센터에서 로그인 상태 최종 확인...")
		if b.CheckLoginStatus() {
			log.Println("🎉 로그인 상태 재확인 완료 - 이미 로그인됨")
			b.isLoggedIn = true
			return nil
		}
		
		// 로그인이 필요한 경우 OAuth 페이지로 이동
		log.Println("🔄 로그인 페이지로 이동 필요")
		oauthURL := b.baseURL + "/oauth2/authorization/gcdm?language=ko"
		log.Printf("OAuth URL로 이동: %s", oauthURL)
		
		_, err := b.page.Goto(oauthURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout: playwright.Float(15000), // 30초에서 15초로 단축
		})
		if err != nil {
			// Timeout is expected due to redirects
			log.Printf("⚠️ 페이지 이동 경고 (정상): %v", err)
		}
		
		// Wait for redirect to BMW login page - 스마트 대기
		log.Println("BMW 로그인 페이지 리다이렉트 대기...")
		for i := 0; i < 10; i++ { // 최대 10초 대기
			time.Sleep(500 * time.Millisecond)
			currentURL = b.page.URL()
			if strings.Contains(currentURL, "customer.bmwgroup.com") {
				log.Printf("✅ 리다이렉트 완료: %s", currentURL)
				break
			}
		}
		currentURL = b.page.URL()
		log.Printf("📍 리다이렉트 후 URL: %s", currentURL)
		
		if !strings.Contains(currentURL, "customer.bmwgroup.com") {
			return fmt.Errorf("로그인 페이지로 이동 실패: %s", currentURL)
		}
	}
	
	log.Println("✅ BMW 고객 계정 로그인 페이지 감지")
	log.Println("⚡ 로그인 준비 중...")
	
	// Angular 앱이 완전히 로드될 때까지 대기 (필수!)
	log.Println("   Angular 앱 로딩 대기 (3초)...")
	time.Sleep(3 * time.Second)
	
	// BMW 로그인 페이지의 정확한 이메일 필드 선택
	log.Println("🔍 이메일 필드 찾는 중...")
	emailField := b.page.Locator("input#email")
	
	// 이메일 필드가 나타날 때까지 대기
	err := emailField.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(5000), // 5초 대기
	})
	if err != nil {
		log.Printf("⚠️ #email로 찾기 실패: %v", err)
		// 폴백: name 속성으로 시도
		emailField = b.page.Locator("input[name='email']")
		err = emailField.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(3000),
		})
		if err != nil {
			log.Printf("❌ 이메일 필드를 찾을 수 없음: %v", err)
			// 마지막 시도: type=text 첫 번째 필드
			emailField = b.page.Locator("input[type='text']").First()
		}
	}
	log.Println("✅ 이메일 필드 발견")
	
	// hCaptcha 감지
	log.Println("🛡️ hCaptcha 확인 중...")
	captchaFrame := b.page.Locator("iframe[src*='hcaptcha']")
	captchaCount, _ := captchaFrame.Count()
	if captchaCount > 0 {
		log.Println("🚨🚨🚨 hCaptcha 감지됨! 🚨🚨🚨")
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Println("⚠️  CAPTCHA가 표시되었습니다!")
		log.Println("⚠️  브라우저에서 수동으로 CAPTCHA를 완료해주세요.")
		log.Println("⚠️  완료 후 Enter 키를 눌러 계속 진행하세요.")
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		
		// 사용자 입력 대기
		fmt.Print("CAPTCHA 완료 후 Enter 키를 누르세요...")
		fmt.Scanln()
		log.Println("✅ 사용자 확인 완료, 로그인 계속 진행...")
	}
	
	// ==== STEP 1: 이메일 입력 ====
	log.Println("\n===== STEP 1: 이메일 입력 =====")
	// emailField는 이미 위에서 정의되었으므로 재사용
	
	// 필드가 실제로 존재하는지 확인
	count, _ := emailField.Count()
	if count == 0 {
		return fmt.Errorf("이메일 필드가 페이지에 없음")
	}
	
	log.Println("이메일 필드 클릭...")
	err = emailField.Click()
	if err != nil {
		log.Printf("⚠️ 클릭 실패: %v", err)
	}
	
	// 클릭 후 약간 대기
	time.Sleep(300 * time.Millisecond)
	
	log.Printf("이메일 입력: %s", username)
	// Type이 더 안정적
	err = emailField.Type(username, playwright.LocatorTypeOptions{
		Delay: playwright.Float(50),
	})
	if err != nil {
		log.Printf("⚠️ Type 실패, Fill 시도: %v", err)
		err = emailField.Fill(username)
		if err != nil {
			return fmt.Errorf("이메일 입력 실패: %w", err)
		}
	}
	log.Println("✅ 이메일 입력 완료")
	
	// ==== STEP 2: "계속" 버튼 클릭 (비밀번호 화면으로 이동) ====
	log.Println("\n===== STEP 2: '계속' 버튼 클릭 =====")
	
	// 이메일 입력 후 버튼 활성화 대기
	time.Sleep(500 * time.Millisecond)
	
	// BMW 페이지의 정확한 계속 버튼 선택
	continueButton := b.page.Locator("button.custom-button.primary").First()
	log.Println("   🔘 계속 버튼 활성화 대기...")
	
	// 버튼이 disabled 상태에서 enabled로 변할 때까지 대기
	for i := 0; i < 30; i++ { // 최대 3초 대기
		disabled, _ := continueButton.GetAttribute("disabled")
		if disabled == "" || disabled == "false" {
			log.Println("✅ 버튼 활성화됨")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	log.Println("버튼 클릭...")
	err = continueButton.Click()
	if err != nil {
		log.Printf("⚠️ 클릭 실패: %v", err)
		// 폴백: 텍스트로 찾기
		continueButton = b.page.Locator("button:has-text('계속')").First()
		err = continueButton.Click()
		if err != nil {
			log.Printf("⚠️ 계속 버튼 클릭 실패: %v", err)
		}
	} else {
		log.Println("✅ 버튼 클릭 성공")
	}
	
	// Wait for password screen to load by waiting for password field
	log.Println("비밀번호 필드 로딩 대기...")
	passwordField := b.page.Locator("input#password")
	err = passwordField.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(8000),
	})
	if err != nil {
		log.Printf("⚠️ 비밀번호 필드 대기 실패: %v", err)
		// 폴백: name 속성으로 시도
		passwordField = b.page.Locator("input[name='password']")
		err = passwordField.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(3000),
		})
		if err != nil {
			log.Printf("⚠️ 비밀번호 필드를 찾을 수 없음: %v", err)
		}
	} else {
		log.Println("✅ 비밀번호 필드 준비 완료")
	}
	
	// ==== STEP 3: 비밀번호 입력 ====
	log.Println("\n===== STEP 3: 비밀번호 입력 =====")
	// passwordField는 이미 위에서 WaitFor로 확인했으므로 재사용
	
	log.Println("비밀번호 필드 클릭...")
	err = passwordField.Click()
	if err != nil {
		log.Printf("⚠️ 클릭 실패: %v", err)
	}
	
	// 클릭 후 약간 대기
	time.Sleep(300 * time.Millisecond)
	
	log.Println("비밀번호 입력...")
	// Type이 더 안정적
	err = passwordField.Type(password, playwright.LocatorTypeOptions{
		Delay: playwright.Float(50),
	})
	if err != nil {
		log.Printf("⚠️ Type 실패, Fill 시도: %v", err)
		err = passwordField.Fill(password)
		if err != nil {
			return fmt.Errorf("비밀번호 입력 실패: %w", err)
		}
	}
	log.Println("✅ 비밀번호 입력 완료")
	
	// ==== STEP 4: 최종 로그인 버튼 클릭 ====
	log.Println("\n===== STEP 4: 최종 로그인 =====")
	
	// 비밀번호 입력 후 버튼 활성화 대기
	time.Sleep(500 * time.Millisecond)
	
	// BMW 페이지의 정확한 로그인 버튼 선택 (비밀번호 화면에서는 '로그인' 텍스트)
	finalButton := b.page.Locator("button.custom-button.primary").First()
	log.Println("   🔘 로그인 버튼 활성화 대기...")
	
	// 버튼이 disabled 상태에서 enabled로 변할 때까지 대기
	for i := 0; i < 30; i++ { // 최대 3초 대기
		disabled, _ := finalButton.GetAttribute("disabled")
		if disabled == "" || disabled == "false" {
			log.Println("✅ 버튼 활성화됨")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	log.Println("로그인 버튼 클릭...")
	err = finalButton.Click()
	if err != nil {
		log.Printf("⚠️ 클릭 실패: %v", err)
		// 폴백: 텍스트로 찾기
		finalButton = b.page.Locator("button:has-text('로그인')").First()
		err = finalButton.Click()
		if err != nil {
			log.Printf("⚠️ 로그인 버튼 클릭 실패, Enter 키 시도: %v", err)
			b.page.Keyboard().Press("Enter")
		}
	} else {
		log.Println("✅ 로그인 버튼 클릭 성공")
	}
	
	// ==== 로그인 처리 대기 ====
	log.Println("\n===== 로그인 처리 대기 =====")
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		currentURL := b.page.URL()
		log.Printf("[%d초] 현재 URL: %s", i+1, currentURL)
		
		if strings.Contains(currentURL, "driving-center.bmw.co.kr") {
			log.Println("\n🎉🎉 로그인 성공! BMW 드라이빙 센터로 리다이렉트됨 🎉🎉")
			return nil
		}
		
		// Check for errors and hCaptcha periodically
		if i == 5 || i == 10 {
			// Check for hCaptcha during login process
			captchaFrame := b.page.Locator("iframe[src*='hcaptcha']")
			captchaCount, _ := captchaFrame.Count()
			if captchaCount > 0 {
				log.Println("\n🚨🚨🚨 로그인 중 hCaptcha 감지됨! 🚨🚨🚨")
				log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				log.Println("⚠️  CAPTCHA가 표시되었습니다!")
				log.Println("⚠️  브라우저에서 수동으로 CAPTCHA를 완료해주세요.")
				log.Println("⚠️  완료 후 Enter 키를 눌러 계속 진행하세요.")
				log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				
				// 사용자 입력 대기
				fmt.Print("CAPTCHA 완료 후 Enter 키를 누르세요...")
				fmt.Scanln()
				log.Println("✅ 사용자 확인 완료, 로그인 계속 진행...")
				continue
			}
			
			// Check for login errors
			errorMsg := b.page.Locator(".error, .alert, [role='alert']")
			if errorCount, _ := errorMsg.Count(); errorCount > 0 {
				errorText, _ := errorMsg.First().TextContent()
				if errorText != "" {
					return fmt.Errorf("로그인 실패: %s", strings.TrimSpace(errorText))
				}
			}
		}
	}
	
	// Final check
	finalURL := b.page.URL()
	if !strings.Contains(finalURL, "driving-center.bmw.co.kr") {
		log.Println("\n❌ 로그인 실패 - 타임아웃")
		return fmt.Errorf("로그인 실패 - 아이디/비밀번호를 확인해주세요")
	}
	
	// 로그인 성공 후 세션 저장
	b.isLoggedIn = true
	saveErr := b.SaveSession()
	if saveErr != nil {
		log.Printf("⚠️ 세션 저장 실패: %v", saveErr)
	}
	
	return nil
}

// CheckReservationPage checks the reservation page for available programs
func (b *BrowserClient) CheckReservationPage(programs []string) (map[string]bool, error) {
	// Navigate to reservation page
	_, err := b.page.Goto(b.baseURL + "/orders/programs/products/view")
	if err != nil {
		return nil, fmt.Errorf("예약 페이지 이동 실패 (failed to navigate to reservation page): %w", err)
	}

	// Wait for page to load
	time.Sleep(3 * time.Second)

	// Get page content
	content, err := b.page.Content()
	if err != nil {
		return nil, fmt.Errorf("페이지 내용 가져오기 실패 (failed to get page content): %w", err)
	}

	result := make(map[string]bool)
	for _, program := range programs {
		// Check if program exists and is available
		if contains(content, program) {
			// Check if it's sold out
			isSoldOut := contains(content, program+".*매진") || contains(content, program+".*마감")
			result[program] = !isSoldOut
		} else {
			result[program] = false
		}
	}

	return result, nil
}

// Close closes the browser
func (b *BrowserClient) Close() error {
	if b.page != nil {
		b.page.Close()
	}
	if b.context != nil {
		b.context.Close()
	}
	if b.browser != nil {
		b.browser.Close()
	}
	if b.pw != nil {
		b.pw.Stop()
	}
	return nil
}

// contains is a simple string contains function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || contains(s[1:], substr))))
}