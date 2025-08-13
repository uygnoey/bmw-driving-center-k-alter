package browser

import (
	"bmw-driving-center-alter/internal/config"
	"bmw-driving-center-alter/internal/solver"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

// BrowserClient handles browser-based authentication and scraping using Selenium
type BrowserClient struct {
	driver           selenium.WebDriver
	service          *selenium.Service
	baseURL          string
	stateDir         string
	isLoggedIn       bool
	captchaSolver    solver.HCaptchaSolver
	autoSolveCaptcha bool
}

// NewBrowserClient creates a new browser client with Selenium
func NewBrowserClient() (*BrowserClient, error) {
	return NewBrowserClientWithConfig(nil)
}

// NewBrowserClientWithConfig creates a new browser client with configuration
func NewBrowserClientWithConfig(cfg *config.Config) (*BrowserClient, error) {
	// 세션 저장 디렉토리 설정
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".bmw-driving-center", "browser-state")
	
	// 디렉토리 생성
	err := os.MkdirAll(stateDir, 0755)
	if err != nil {
		log.Printf("⚠️ 세션 디렉토리 생성 실패: %v", err)
	}

	client := &BrowserClient{
		baseURL:    "https://driving-center.bmw.co.kr",
		stateDir:   stateDir,
		isLoggedIn: false,
		autoSolveCaptcha: false,
	}
	
	// Check config first, then environment variables
	var apiKey string
	var service string
	
	if cfg != nil && cfg.CaptchaSolver.APIKey != "" {
		// Use config settings
		apiKey = cfg.CaptchaSolver.APIKey
		service = cfg.CaptchaSolver.Service
		if service == "" {
			service = "solvecaptcha" // default to solvecaptcha
		}
	} else {
		// Check environment variables
		if key := os.Getenv("SOLVECAPTCHA_API_KEY"); key != "" {
			apiKey = key
			service = "solvecaptcha"
		} else if key := os.Getenv("TWOCAPTCHA_API_KEY"); key != "" {
			apiKey = key
			service = "2captcha"
		}
	}
	
	// Setup captcha solver
	if apiKey != "" {
		switch service {
		case "solvecaptcha":
			log.Println("🤖 SolveCaptcha 자동 hCaptcha 해결 활성화")
			client.captchaSolver = solver.NewSolveCaptchaSolver(apiKey)
			client.autoSolveCaptcha = true
		case "2captcha":
			log.Println("🤖 2captcha 자동 hCaptcha 해결 활성화")
			client.captchaSolver = solver.NewTwoCaptchaSolver(apiKey)
			client.autoSolveCaptcha = true
		default:
			log.Printf("⚠️ 알 수 없는 captcha solver 서비스: %s", service)
			client.captchaSolver = solver.NewManualSolver()
		}
	} else {
		log.Println("🔑 Captcha solver API 키 없음 - 수동 hCaptcha 해결 모드")
		log.Println("💡 자동 해결을 원하면 config.yaml에 설정하거나:")
		log.Println("   - SolveCaptcha: export SOLVECAPTCHA_API_KEY=your_api_key")
		log.Println("   - 2captcha: export TWOCAPTCHA_API_KEY=your_api_key")
		client.captchaSolver = solver.NewManualSolver()
	}
	
	return client, nil
}

// downloadChromeDriver downloads the latest ChromeDriver if needed
func (b *BrowserClient) downloadChromeDriver() (string, error) {
	driverDir := filepath.Join(b.stateDir, "drivers")
	os.MkdirAll(driverDir, 0755)
	
	// OS별 ChromeDriver 파일명
	driverName := "chromedriver"
	if runtime.GOOS == "windows" {
		driverName = "chromedriver.exe"
	}
	driverPath := filepath.Join(driverDir, driverName)
	
	// 이미 존재하면 사용
	if _, err := os.Stat(driverPath); err == nil {
		log.Printf("✅ ChromeDriver 이미 존재: %s", driverPath)
		return driverPath, nil
	}
	
	log.Println("📥 ChromeDriver 다운로드 중...")
	
	// Chrome 버전 확인
	chromeVersion, err := getChromeVersion()
	if err != nil {
		log.Printf("⚠️ Chrome 버전 확인 실패: %v", err)
		chromeVersion = "stable"
	}
	
	// ChromeDriver 다운로드 URL 생성
	downloadURL := getChromeDriverURL(chromeVersion)
	log.Printf("   다운로드 URL: %s", downloadURL)
	
	// 다운로드
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("ChromeDriver 다운로드 실패: %w", err)
	}
	defer resp.Body.Close()
	
	// ZIP 파일로 저장
	zipFile := filepath.Join(driverDir, "chromedriver.zip")
	out, err := os.Create(zipFile)
	if err != nil {
		return "", fmt.Errorf("파일 생성 실패: %w", err)
	}
	
	size, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return "", fmt.Errorf("파일 저장 실패: %w", err)
	}
	log.Printf("   다운로드 완료: %d bytes", size)
	
	// 압축 해제
	log.Println("   압축 해제 중...")
	cmd := exec.Command("unzip", "-o", zipFile, "-d", driverDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("압축 해제 실패: %w\n출력: %s", err, string(output))
	}
	log.Printf("   압축 해제 완료: %s", string(output))
	
	// ZIP 파일 삭제
	os.Remove(zipFile)
	
	// chromedriver-mac-arm64/chromedriver 같은 하위 디렉토리 확인
	entries, err := os.ReadDir(driverDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), "chromedriver") {
				// 하위 디렉토리의 chromedriver를 상위로 이동
				subDriverPath := filepath.Join(driverDir, entry.Name(), "chromedriver")
				if _, err := os.Stat(subDriverPath); err == nil {
					log.Printf("   ChromeDriver 발견: %s", subDriverPath)
					os.Rename(subDriverPath, driverPath)
					os.RemoveAll(filepath.Join(driverDir, entry.Name()))
					break
				}
			}
		}
	}
	
	// 실행 권한 부여 (Unix 계열)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(driverPath, 0755); err != nil {
			log.Printf("⚠️ 실행 권한 설정 실패: %v", err)
		}
	}
	
	log.Printf("✅ ChromeDriver 다운로드 완료: %s", driverPath)
	return driverPath, nil
}

// getChromeVersion gets the installed Chrome version
func getChromeVersion() (string, error) {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--version")
	case "linux":
		cmd = exec.Command("google-chrome", "--version")
	case "windows":
		cmd = exec.Command("cmd", "/c", `reg query "HKEY_CURRENT_USER\Software\Google\Chrome\BLBeacon" /v version`)
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	// 버전 파싱 (예: "Google Chrome 139.0.6812.86" -> "139")
	versionStr := string(output)
	parts := strings.Fields(versionStr)
	for _, part := range parts {
		if strings.Contains(part, ".") {
			versionParts := strings.Split(part, ".")
			if len(versionParts) > 0 {
				return versionParts[0], nil
			}
		}
	}
	
	return "", fmt.Errorf("Chrome 버전을 파싱할 수 없음")
}

// getChromeDriverURL returns the download URL for ChromeDriver
func getChromeDriverURL(chromeVersion string) string {
	log.Printf("   Chrome 버전 %s에 맞는 ChromeDriver 검색 중...", chromeVersion)
	
	// Chrome for Testing API 사용
	apiURL := "https://googlechromelabs.github.io/chrome-for-testing/known-good-versions-with-downloads.json"
	
	// API 호출
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("⚠️ ChromeDriver API 호출 실패: %v", err)
		// 폴백 URL 반환
		return getStableChromeDriverURL()
	}
	defer resp.Body.Close()
	
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("⚠️ API 응답 파싱 실패: %v", err)
		return getStableChromeDriverURL()
	}
	
	// Chrome 버전에 맞는 버전 찾기
	versions, ok := data["versions"].([]interface{})
	if !ok || len(versions) == 0 {
		return getStableChromeDriverURL()
	}
	
	// Chrome 버전과 일치하는 ChromeDriver 찾기
	platform := getPlatform()
	var bestMatch map[string]interface{}
	
	for _, v := range versions {
		version := v.(map[string]interface{})
		versionStr, ok := version["version"].(string)
		if !ok {
			continue
		}
		
		// 메이저 버전 비교 (139.x.x.x -> 139)
		if strings.HasPrefix(versionStr, chromeVersion+".") {
			// downloads 확인
			if downloads, ok := version["downloads"].(map[string]interface{}); ok {
				if chromedriver, ok := downloads["chromedriver"].([]interface{}); ok && len(chromedriver) > 0 {
					bestMatch = version
					// 가장 최신 버전 사용을 위해 계속 검색
				}
			}
		}
	}
	
	// 일치하는 버전 찾음
	if bestMatch != nil {
		log.Printf("   ChromeDriver 버전 발견: %s", bestMatch["version"])
		downloads := bestMatch["downloads"].(map[string]interface{})
		chromedriver := downloads["chromedriver"].([]interface{})
		
		// OS별 URL 찾기
		for _, item := range chromedriver {
			download := item.(map[string]interface{})
			if download["platform"] == platform {
				url := download["url"].(string)
				log.Printf("   다운로드 URL: %s", url)
				return url
			}
		}
	}
	
	// 정확한 버전을 찾지 못한 경우 가장 가까운 버전 사용
	log.Printf("⚠️ Chrome %s용 정확한 ChromeDriver를 찾지 못함, 대체 버전 사용", chromeVersion)
	
	// Chrome 139용 직접 URL (하드코딩)
	if chromeVersion == "139" {
		baseURL := "https://storage.googleapis.com/chrome-for-testing-public/139.0.6812.58"
		url := fmt.Sprintf("%s/%s/chromedriver-%s.zip", baseURL, platform, platform)
		log.Printf("   Chrome 139용 대체 URL: %s", url)
		return url
	}
	
	return getStableChromeDriverURL()
}

func getPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-arm64"
		}
		return "mac-x64"
	case "linux":
		return "linux64"
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "win64"
		}
		return "win32"
	default:
		return "linux64"
	}
}

func getStableChromeDriverURL() string {
	platform := getPlatform()
	// 최신 stable 버전 URL (수동 업데이트 필요)
	baseURL := "https://storage.googleapis.com/chrome-for-testing-public/139.0.6812.86"
	return fmt.Sprintf("%s/%s/chromedriver-%s.zip", baseURL, platform, platform)
}

// Start launches the browser with Selenium
func (b *BrowserClient) Start(headless bool) error {
	// ChromeDriver 다운로드/확인
	driverPath, err := b.downloadChromeDriver()
	if err != nil {
		log.Printf("⚠️ ChromeDriver 자동 다운로드 실패: %v", err)
		log.Println("수동으로 ChromeDriver를 설치해주세요: https://chromedriver.chromium.org/")
		// 시스템 PATH에서 찾기 시도
		driverPath = "chromedriver"
	}
	
	// Selenium 서비스 시작
	seleniumPath := driverPath // ChromeDriver 경로 사용
	port := 9515
	
	opts := []selenium.ServiceOption{
		selenium.Output(nil), // 로그 비활성화
	}
	
	service, err := selenium.NewChromeDriverService(seleniumPath, port, opts...)
	if err != nil {
		return fmt.Errorf("ChromeDriver 서비스 시작 실패: %w", err)
	}
	b.service = service
	
	// Chrome 옵션 설정 (Stealth 모드)
	chromeCaps := chrome.Capabilities{
		Args: []string{
			// 자동화 감지 회피
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
			"--disable-automation",
			"--disable-infobars",
			
			// 성능 및 안정성
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-gpu",
			"--disable-web-security",
			"--disable-features=VizDisplayCompositor",
			"--disable-background-timer-throttling",
			"--disable-backgrounding-occluded-windows",
			"--disable-renderer-backgrounding",
			
			// 창 설정
			"--window-size=1920,1080",
			"--start-maximized",
			
			// User-Agent (실제 Chrome과 동일하게)
			"--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
			
			// 언어 설정
			"--lang=ko-KR",
			"--accept-lang=ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7",
		},
		Prefs: map[string]interface{}{
			// 자동화 관련 설정 비활성화
			"credentials_enable_service": false,
			"profile.password_manager_enabled": false,
			"profile.default_content_setting_values.notifications": 2,
			"excludeSwitches": []string{"enable-automation"},
			"useAutomationExtension": false,
			
			// WebRTC IP 누출 방지
			"webrtc.ip_handling_policy": "default_public_interface_only",
			"webrtc.multiple_routes_enabled": false,
			"webrtc.nonproxied_udp_enabled": false,
		},
		W3C: false, // W3C 모드 비활성화 (레거시 모드 사용)
	}
	
	if headless {
		chromeCaps.Args = append(chromeCaps.Args, "--headless=new")
	}
	
	// 사용자 데이터 디렉토리 설정 (세션 유지)
	userDataDir := filepath.Join(b.stateDir, "chrome-profile")
	os.MkdirAll(userDataDir, 0755)
	chromeCaps.Args = append(chromeCaps.Args, fmt.Sprintf("--user-data-dir=%s", userDataDir))
	
	caps := selenium.Capabilities{"browserName": "chrome"}
	caps.AddChrome(chromeCaps)
	
	// WebDriver 생성
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		return fmt.Errorf("WebDriver 생성 실패: %w", err)
	}
	b.driver = wd
	
	// JavaScript로 WebDriver 속성 제거 (더 강력한 Stealth)
	script := `
		// WebDriver 속성 완전 제거
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});
		
		// Chrome 속성 실제와 동일하게
		window.chrome = {
			runtime: {},
			loadTimes: function() {},
			csi: function() {}
		};
		
		// 플러그인 배열 실제 Chrome과 동일하게
		Object.defineProperty(navigator, 'plugins', {
			get: () => {
				const PluginArray = function() {};
				const pluginArray = new PluginArray();
				pluginArray[0] = {
					name: 'Chrome PDF Plugin',
					filename: 'internal-pdf-viewer',
					description: 'Portable Document Format'
				};
				pluginArray.length = 1;
				return pluginArray;
			}
		});
		
		// 언어 설정
		Object.defineProperty(navigator, 'languages', {
			get: () => ['ko-KR', 'ko', 'en-US', 'en'],
		});
		
		// 하드웨어 동시성
		Object.defineProperty(navigator, 'hardwareConcurrency', {
			get: () => 8
		});
		
		// 플랫폼
		Object.defineProperty(navigator, 'platform', {
			get: () => 'MacIntel'
		});
		
		// Permission API 수정
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
		
		// WebGL Vendor
		const getParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(parameter) {
			if (parameter === 37445) {
				return 'Intel Inc.';
			}
			if (parameter === 37446) {
				return 'Intel Iris OpenGL Engine';
			}
			return getParameter(parameter);
		};
		
		// Console 수정 (자동화 감지 회피)
		const originalLog = console.log;
		console.log = function() {
			if (arguments[0] && arguments[0].toString().indexOf('webdriver') === -1) {
				return originalLog.apply(console, arguments);
			}
		};
	`
	if _, err := b.driver.ExecuteScript(script, nil); err != nil {
		log.Printf("⚠️ Stealth 스크립트 실행 실패: %v", err)
	}
	
	log.Println("✅ Selenium WebDriver 시작 완료")
	return nil
}

// CheckLoginStatus checks if already logged in
func (b *BrowserClient) CheckLoginStatus() bool {
	log.Println("🔍 로그인 상태 확인 중...")
	
	// 메인 페이지로 이동
	log.Printf("1️⃣ BMW 드라이빙 센터 메인 페이지 접속: %s", b.baseURL)
	if err := b.driver.Get(b.baseURL); err != nil {
		log.Printf("⚠️ 메인 페이지 접속 실패: %v", err)
		return false
	}
	
	time.Sleep(2 * time.Second)
	
	// 예약 페이지로 이동 시도
	log.Println("2️⃣ 예약 페이지로 이동 시도...")
	if err := b.driver.Get(b.baseURL + "/orders/programs/products/view"); err != nil {
		log.Printf("⚠️ 예약 페이지 이동 실패: %v", err)
		return false
	}
	
	time.Sleep(2 * time.Second)
	
	// 현재 URL 확인
	currentURL, _ := b.driver.CurrentURL()
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
	
	log.Printf("⚠️ 예상치 못한 페이지: %s", currentURL)
	b.isLoggedIn = false
	return false
}

// Login performs login to BMW Driving Center
func (b *BrowserClient) Login(username, password string) error {
	log.Println("===== BMW 드라이빙 센터 로그인 시작 =====")
	
	// 현재 페이지 URL 확인
	currentURL, _ := b.driver.CurrentURL()
	log.Printf("📍 현재 페이지: %s", currentURL)
	
	// 로그인 페이지가 아니면 이동
	if !strings.Contains(currentURL, "customer.bmwgroup.com") {
		// 로그인 상태 재확인
		if b.CheckLoginStatus() {
			log.Println("🎉 이미 로그인됨")
			return nil
		}
		
		// OAuth 로그인 페이지로 이동
		oauthURL := b.baseURL + "/oauth2/authorization/gcdm?language=ko"
		log.Printf("OAuth URL로 이동: %s", oauthURL)
		if err := b.driver.Get(oauthURL); err != nil {
			return fmt.Errorf("OAuth 페이지 이동 실패: %w", err)
		}
		
		// 리다이렉트 대기
		time.Sleep(3 * time.Second)
		currentURL, _ = b.driver.CurrentURL()
		log.Printf("📍 리다이렉트 후 URL: %s", currentURL)
	}
	
	log.Println("✅ BMW 고객 계정 로그인 페이지 감지")
	
	// 쿠키 확인
	cookies, _ := b.driver.GetCookies()
	log.Printf("🍪 현재 쿠키 개수: %d", len(cookies))
	
	// localStorage 확인
	if storedParams, err := b.driver.ExecuteScript(`
		return localStorage.getItem('storedParameters');
	`, nil); err == nil && storedParams != nil {
		log.Printf("📦 localStorage.storedParameters: %v", storedParams)
	}
	
	// ==== STEP 1: 이메일 입력 ====
	log.Println("\n===== STEP 1: 이메일 입력 =====")
	
	// 이메일 필드 찾기 (visible만)
	emailField, err := b.driver.FindElement(selenium.ByCSSSelector, "input#email:not([type='hidden'])")
	if err != nil {
		// 폴백: name으로 찾기
		emailField, err = b.driver.FindElement(selenium.ByName, "email")
		if err != nil {
			return fmt.Errorf("이메일 필드를 찾을 수 없음: %w", err)
		}
	}
	
	// 이메일 입력
	log.Printf("이메일 입력: %s", username)
	if err := emailField.Clear(); err != nil {
		log.Printf("⚠️ 필드 클리어 실패: %v", err)
	}
	if err := emailField.SendKeys(username); err != nil {
		return fmt.Errorf("이메일 입력 실패: %w", err)
	}
	log.Println("✅ 이메일 입력 완료")
	
	// ==== STEP 2: "계속" 버튼 클릭 ====
	log.Println("\n===== STEP 2: '계속' 버튼 클릭 =====")
	
	time.Sleep(1 * time.Second)
	
	// 계속 버튼 찾기
	continueBtn, err := b.driver.FindElement(selenium.ByCSSSelector, "button.custom-button.primary")
	if err != nil {
		continueBtn, err = b.driver.FindElement(selenium.ByXPATH, "//button[contains(text(), '계속')]")
		if err != nil {
			return fmt.Errorf("계속 버튼을 찾을 수 없음: %w", err)
		}
	}
	
	// 버튼 클릭
	log.Println("버튼 클릭...")
	if err := continueBtn.Click(); err != nil {
		return fmt.Errorf("계속 버튼 클릭 실패: %w", err)
	}
	log.Println("✅ 버튼 클릭 성공")
	
	// 비밀번호 화면 대기
	time.Sleep(2 * time.Second)
	
	// ==== STEP 3: 비밀번호 입력 ====
	log.Println("\n===== STEP 3: 비밀번호 입력 =====")
	
	// 비밀번호 필드 찾기
	passwordField, err := b.driver.FindElement(selenium.ByCSSSelector, "input#password:not([type='hidden'])")
	if err != nil {
		passwordField, err = b.driver.FindElement(selenium.ByName, "password")
		if err != nil {
			return fmt.Errorf("비밀번호 필드를 찾을 수 없음: %w", err)
		}
	}
	
	// 비밀번호 입력
	log.Println("비밀번호 입력...")
	if err := passwordField.Clear(); err != nil {
		log.Printf("⚠️ 필드 클리어 실패: %v", err)
	}
	if err := passwordField.SendKeys(password); err != nil {
		return fmt.Errorf("비밀번호 입력 실패: %w", err)
	}
	log.Println("✅ 비밀번호 입력 완료")
	
	// ==== STEP 4: 로그인 버튼 클릭 ====
	log.Println("\n===== STEP 4: 로그인 버튼 클릭 =====")
	
	time.Sleep(1 * time.Second)
	
	// 로그인 버튼 찾기
	loginBtn, err := b.driver.FindElement(selenium.ByCSSSelector, "button.custom-button.primary")
	if err != nil {
		loginBtn, err = b.driver.FindElement(selenium.ByXPATH, "//button[contains(text(), '로그인')]")
		if err != nil {
			return fmt.Errorf("로그인 버튼을 찾을 수 없음: %w", err)
		}
	}
	
	// 버튼 클릭
	log.Println("로그인 버튼 클릭...")
	if err := loginBtn.Click(); err != nil {
		return fmt.Errorf("로그인 버튼 클릭 실패: %w", err)
	}
	log.Println("✅ 로그인 버튼 클릭 성공")
	
	// ==== 로그인 처리 대기 ====
	log.Println("\n===== 로그인 처리 대기 =====")
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		currentURL, _ := b.driver.CurrentURL()
		log.Printf("[%d초] 현재 URL: %s", i+1, currentURL)
		
		if strings.Contains(currentURL, "driving-center.bmw.co.kr") {
			log.Println("\n🎉🎉 로그인 성공! BMW 드라이빙 센터로 리다이렉트됨 🎉🎉")
			
			// 로그인 직후 바로 CAPTCHA 확인!!! (아무것도 하지 않고)
			log.Println("\n🔍 로그인 직후 즉시 CAPTCHA 확인 중...")
			time.Sleep(2 * time.Second) // 페이지 안정화를 위한 최소 대기
			
			if b.checkForCaptcha() {
				log.Println("\n🚨🚨🚨 로그인 직후 hCAPTCHA 감지됨! 🚨🚨🚨")
				log.Println("⚠️ CAPTCHA를 먼저 해결해야 합니다!")
				
				// CAPTCHA 해결 대기
				if !b.waitForCaptchaSolution(300) { // 5분 대기
					return fmt.Errorf("로그인 후 CAPTCHA 해결 실패")
				}
				log.Println("✅ CAPTCHA 해결 완료!")
			} else {
				log.Println("✅ CAPTCHA 없음 - 정상 진행")
			}
			
			// CAPTCHA 처리 후에만 다른 작업 수행
			// 로그인 후 쿠키 확인
			cookies, _ := b.driver.GetCookies()
			log.Printf("🍪 로그인 후 쿠키 개수: %d", len(cookies))
			
			// 메인 페이지로 이동하여 세션 안정화
			log.Println("🏠 메인 페이지로 이동하여 세션 확인...")
			if err := b.driver.Get(b.baseURL); err != nil {
				log.Printf("⚠️ 메인 페이지 이동 실패: %v", err)
			}
			time.Sleep(2 * time.Second)
			
			b.isLoggedIn = true
			return nil
		}
		
		// hCaptcha 확인
		if i == 5 || i == 10 {
			if b.checkForCaptcha() {
				log.Println("⏳ CAPTCHA 해결 대기 중...")
				// CAPTCHA가 사라질 때까지 추가 대기
				for j := 0; j < 30; j++ {
					time.Sleep(1 * time.Second)
					if !b.checkForCaptcha() {
						log.Println("✅ CAPTCHA 해결됨")
						break
					}
				}
			}
		}
	}
	
	return fmt.Errorf("로그인 실패 - 타임아웃")
}

// checkForCaptcha checks if hCaptcha is present
func (b *BrowserClient) checkForCaptcha() bool {
	// nil 체크
	if b.driver == nil {
		return false
	}
	
	captchaDetected := false
	
	// 1. 페이지 소스 전체에서 hCaptcha 관련 문자열 확인
	pageSource, err := b.driver.PageSource()
	if err != nil {
		return false
	}
	
	// 페이지가 거의 비어있고 hcaptcha 관련 내용이 있는지 확인
	// hCaptcha 페이지는 보통 매우 작음 (10KB 이하)
	if len(pageSource) < 10000 {
		// hCaptcha 관련 문자열들 확인
		if strings.Contains(pageSource, "hcaptcha.com") ||
		   strings.Contains(pageSource, "h-captcha") ||
		   strings.Contains(pageSource, "hCaptcha") ||
		   strings.Contains(pageSource, "https://hcaptcha.com/license") {
			captchaDetected = true
		}
	}
	
	// 2. body class="no-selection" 체크 (가장 중요한 지표)
	if !captchaDetected {
		// body의 innerHTML이 hcaptcha 관련 내용으로 시작하는지 체크
		bodyHTML, err := b.driver.FindElement(selenium.ByTagName, "body")
		if err == nil {
			className, _ := bodyHTML.GetAttribute("class")
			innerHTML, _ := bodyHTML.GetAttribute("innerHTML")
			
			// no-selection 클래스 확인
			if className == "no-selection" || strings.Contains(className, "no-selection") {
				captchaDetected = true
			}
			
			// innerHTML이 hcaptcha로 시작하거나 포함하는지 확인
			if len(innerHTML) < 5000 && (strings.Contains(innerHTML, "hcaptcha") || 
			                              strings.Contains(innerHTML, "h-captcha")) {
				captchaDetected = true
			}
		}
	}
	
	// 3. iframe 확인
	if !captchaDetected {
		iframes, err := b.driver.FindElements(selenium.ByTagName, "iframe")
		if err == nil {
			for _, iframe := range iframes {
				src, _ := iframe.GetAttribute("src")
				title, _ := iframe.GetAttribute("title")
				if strings.Contains(src, "hcaptcha") || strings.Contains(title, "hCaptcha") ||
				   strings.Contains(src, "newassets.hcaptcha.com") {
					captchaDetected = true
					break
				}
			}
		}
	}
	
	// 4. div class 확인
	if !captchaDetected {
		divs, err := b.driver.FindElements(selenium.ByClassName, "h-captcha")
		if err == nil && len(divs) > 0 {
			captchaDetected = true
		}
	}
	
	// 5. 특정 스크립트 태그 확인
	if !captchaDetected {
		scripts, err := b.driver.FindElements(selenium.ByTagName, "script")
		if err == nil {
			for _, script := range scripts {
				src, _ := script.GetAttribute("src")
				if strings.Contains(src, "hcaptcha.com/1/api.js") {
					captchaDetected = true
					break
				}
			}
		}
	}
	
	// 6. 페이지 타이틀 확인 (hCaptcha 페이지는 종종 특별한 타이틀을 가짐)
	if !captchaDetected {
		title, _ := b.driver.Title()
		if strings.Contains(strings.ToLower(title), "captcha") ||
		   strings.Contains(strings.ToLower(title), "verification") ||
		   title == "" && len(pageSource) < 10000 {
			// 타이틀이 비어있고 페이지가 작으면 의심
			if strings.Contains(pageSource, "hcaptcha") {
				captchaDetected = true
			}
		}
	}
	
	if captchaDetected {
		log.Println("\n")
		log.Println("🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨")
		log.Println("🚨                                                  🚨")
		log.Println("🚨           hCAPTCHA 감지됨!!!                    🚨")
		log.Println("🚨                                                  🚨")
		log.Println("🚨   🖱️  브라우저 창을 확인하세요!                  🚨")
		log.Println("🚨   ✅ CAPTCHA를 수동으로 해결해주세요!           🚨")
		log.Println("🚨   ⏳ 해결 후 자동으로 진행됩니다...             🚨")
		log.Println("🚨                                                  🚨")
		log.Println("🚨   👉 잠시만 기다려주세요!!!                     🚨")
		log.Println("🚨   👉 프로그램이 계속 실행 중입니다!!!           🚨")
		log.Println("🚨   👉 종료하지 마세요!!!                         🚨")
		log.Println("🚨                                                  🚨")
		log.Println("🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨")
		log.Println("\n")
	}
	
	return captchaDetected
}

// waitForCaptchaSolution waits for the captcha to be solved
func (b *BrowserClient) waitForCaptchaSolution(timeoutSeconds int) bool {
	// Try auto-solving first if enabled
	if b.autoSolveCaptcha && b.captchaSolver != nil {
		log.Println("🤖 hCaptcha 자동 해결 시도 중...")
		
		// Extract sitekey from page
		siteKey := b.extractSiteKey()
		if siteKey != "" {
			currentURL, _ := b.driver.CurrentURL()
			solution, err := b.captchaSolver.SolveHCaptcha(siteKey, currentURL)
			
			if err == nil && solution != "" {
				// Inject solution into page
				if b.injectCaptchaSolution(solution) {
					log.Println("✅ hCaptcha 자동 해결 성공!")
					time.Sleep(2 * time.Second)
					return true
				}
			}
			log.Println("⚠️ 자동 해결 실패, 수동 모드로 전환")
		}
	}
	
	// Manual solving fallback
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("⏸️  프로그램 일시 정지 - hCAPTCHA 해결 필요")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("👆 브라우저 창에서 CAPTCHA를 해결해주세요")
	log.Printf("⏱️  최대 %d초간 대기합니다...", timeoutSeconds)
	log.Println("💡 TIP: 체크박스를 클릭하거나 이미지를 선택하세요")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	for i := 0; i < timeoutSeconds; i++ {
		time.Sleep(1 * time.Second)
		
		// 10초마다 상태 체크
		if i%10 == 0 && i > 0 {
			// CAPTCHA가 사라졌는지 확인
			if !b.checkForCaptchaQuiet() {
				log.Println("\n✅ CAPTCHA 해결 완료!")
				time.Sleep(2 * time.Second) // 페이지 전환 대기
				return true
			}
			
			// 진행 상황 표시
			if i%30 == 0 {
				remaining := timeoutSeconds - i
				log.Printf("⏳ CAPTCHA 대기 중... (남은 시간: %d초)", remaining)
			}
		}
	}
	
	log.Printf("\n⏱️ CAPTCHA 해결 시간 초과 (%d초)", timeoutSeconds)
	return false
}

// checkForCaptchaQuiet checks for captcha without logging alerts
func (b *BrowserClient) checkForCaptchaQuiet() bool {
	if b.driver == nil {
		return false
	}
	
	pageSource, err := b.driver.PageSource()
	if err != nil {
		return false
	}
	
	// Quick check for hCaptcha
	if len(pageSource) < 10000 {
		if strings.Contains(pageSource, "hcaptcha.com") ||
		   strings.Contains(pageSource, "h-captcha") ||
		   strings.Contains(pageSource, "hCaptcha") {
			return true
		}
	}
	
	// Check body class
	body, err := b.driver.FindElement(selenium.ByTagName, "body")
	if err == nil {
		className, _ := body.GetAttribute("class")
		if strings.Contains(className, "no-selection") {
			return true
		}
	}
	
	return false
}

// extractSiteKey extracts hCaptcha sitekey from the page
func (b *BrowserClient) extractSiteKey() string {
	// Try to find h-captcha div with data-sitekey
	divs, err := b.driver.FindElements(selenium.ByClassName, "h-captcha")
	if err == nil && len(divs) > 0 {
		for _, div := range divs {
			siteKey, err := div.GetAttribute("data-sitekey")
			if err == nil && siteKey != "" {
				log.Printf("🔑 hCaptcha sitekey 발견: %s", siteKey)
				return siteKey
			}
		}
	}
	
	// Try to find in iframe src
	iframes, err := b.driver.FindElements(selenium.ByTagName, "iframe")
	if err == nil {
		for _, iframe := range iframes {
			src, _ := iframe.GetAttribute("src")
			if strings.Contains(src, "hcaptcha.com/captcha/v1") {
				// Extract sitekey from URL
				if idx := strings.Index(src, "sitekey="); idx != -1 {
					siteKey := src[idx+8:]
					if ampIdx := strings.Index(siteKey, "&"); ampIdx != -1 {
						siteKey = siteKey[:ampIdx]
					}
					log.Printf("🔑 hCaptcha sitekey 발견 (iframe): %s", siteKey)
					return siteKey
				}
			}
		}
	}
	
	log.Println("⚠️ hCaptcha sitekey를 찾을 수 없음")
	return ""
}

// injectCaptchaSolution injects the captcha solution into the page
func (b *BrowserClient) injectCaptchaSolution(token string) bool {
	// Inject the token using JavaScript
	script := fmt.Sprintf(`
		document.querySelector('[name="h-captcha-response"]').value = '%s';
		document.querySelector('[name="g-recaptcha-response"]').value = '%s';
		if (typeof hcaptcha !== 'undefined') {
			hcaptcha.setResponse('%s');
		}
		// Try to submit form
		var forms = document.querySelectorAll('form');
		for (var i = 0; i < forms.length; i++) {
			if (forms[i].querySelector('[name="h-captcha-response"]')) {
				forms[i].submit();
				break;
			}
		}
	`, token, token, token)
	
	_, err := b.driver.ExecuteScript(script, nil)
	if err != nil {
		log.Printf("⚠️ 솔루션 주입 실패: %v", err)
		return false
	}
	
	log.Println("✅ hCaptcha 솔루션 주입 성공")
	return true
}


// CheckReservationPageWithCaptchaAlert checks the reservation page
func (b *BrowserClient) CheckReservationPageWithCaptchaAlert(programs []string) (map[string]bool, bool, error) {
	log.Println("📋 예약 페이지 확인 시작...")
	
	// 현재 URL 확인
	currentURL, _ := b.driver.CurrentURL()
	log.Printf("   현재 URL: %s", currentURL)
	
	// 예약 페이지가 아닌 경우에만 이동
	if !strings.Contains(currentURL, "/orders/programs/products/view") {
		log.Println("📋 예약 페이지로 이동...")
		if err := b.driver.Get(b.baseURL + "/orders/programs/products/view"); err != nil {
			return nil, false, fmt.Errorf("예약 페이지 이동 실패: %w", err)
		}
		
		log.Println("⏳ 페이지 로딩 대기 중... (3초)")
		time.Sleep(3 * time.Second)
	} else {
		// 이미 예약 페이지에 있는 경우 새로고침
		log.Println("🔄 예약 페이지 새로고침...")
		if err := b.driver.Refresh(); err != nil {
			log.Printf("⚠️ 페이지 새로고침 실패: %v", err)
		}
		log.Println("⏳ 페이지 로딩 대기 중... (2초)")
		time.Sleep(2 * time.Second)
	}
	
	// 페이지 로딩 후 URL 다시 확인
	currentURL, _ = b.driver.CurrentURL()
	log.Printf("   이동 후 URL: %s", currentURL)
	
	// 페이지 내용 가져오기
	pageSource, err := b.driver.PageSource()
	if err != nil {
		return nil, false, fmt.Errorf("페이지 내용 가져오기 실패: %w", err)
	}
	
	result := make(map[string]bool)
	for _, program := range programs {
		// 프로그램 존재 및 예약 가능 여부 확인
		if strings.Contains(pageSource, program) {
			// 매진/마감 확인
			isSoldOut := strings.Contains(pageSource, program+".*매진") || 
			            strings.Contains(pageSource, program+".*마감")
			result[program] = !isSoldOut
		} else {
			result[program] = false
		}
	}
	
	// CAPTCHA는 이제 로그인 직후에만 확인하므로 여기서는 false 반환
	return result, false, nil
}

// CheckReservationPage checks the reservation page (backward compatibility)
func (b *BrowserClient) CheckReservationPage(programs []string) (map[string]bool, error) {
	result, _, err := b.CheckReservationPageWithCaptchaAlert(programs)
	return result, err
}

// SaveSession saves the current browser session
func (b *BrowserClient) SaveSession() error {
	// Selenium with Chrome user-data-dir automatically saves session
	log.Println("✅ 세션은 Chrome 프로필에 자동 저장됨")
	return nil
}

// Close closes the browser
func (b *BrowserClient) Close() error {
	if b.driver != nil {
		if err := b.driver.Quit(); err != nil {
			log.Printf("⚠️ WebDriver 종료 오류: %v", err)
		}
	}
	if b.service != nil {
		if err := b.service.Stop(); err != nil {
			log.Printf("⚠️ Selenium 서비스 종료 오류: %v", err)
		}
	}
	return nil
}