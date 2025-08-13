package auth

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// LoginCredentials holds login information
type LoginCredentials struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// AuthClient manages authenticated HTTP requests
type AuthClient struct {
	client      *http.Client
	credentials LoginCredentials
	baseURL     string
	isLoggedIn  bool
}

// NewAuthClient creates a new authenticated client
func NewAuthClient(credentials LoginCredentials) (*AuthClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("쿠키 저장소 생성 실패 (failed to create cookie jar): %w", err)
	}

	return &AuthClient{
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil // Allow redirects
			},
		},
		credentials: credentials,
		baseURL:     "https://driving-center.bmw.co.kr",
		isLoggedIn:  false,
	}, nil
}

// Login performs login to BMW Driving Center
func (a *AuthClient) Login() error {
	// BMW 드라이빙 센터는 OAuth2를 사용합니다
	// 실제로 복잡한 OAuth 플로우를 구현하는 대신,
	// 로그인이 필요한 페이지에 접근 시 자동 리다이렉트를 활용합니다

	log.Printf("BMW 드라이빙 센터 OAuth2 로그인 프로세스 시작...")

	// OAuth2 로그인 URL로 이동
	oauthURL := a.baseURL + "/oauth2/authorization/gcdm?language=ko"

	resp, err := a.client.Get(oauthURL)
	if err != nil {
		return fmt.Errorf("OAuth 로그인 페이지 접근 실패: %w", err)
	}
	defer resp.Body.Close()

	// OAuth 프로세스는 복잡하므로, 실제로는 다음과 같은 대안을 사용합니다:
	// 1. Selenium이나 Playwright 같은 브라우저 자동화 도구 사용
	// 2. 또는 세션 쿠키를 직접 설정

	// 현재는 임시로 성공했다고 가정
	log.Printf("OAuth2 로그인은 브라우저 자동화가 필요합니다. 현재 구현을 위해서는 Selenium 사용을 권장합니다.")

	// 실제 구현을 위한 주석:
	// BMW 드라이빙 센터는 BMW 그룹의 통합 인증 시스템(GCDM)을 사용합니다
	// 이는 OAuth2 기반이며, 실제 로그인을 위해서는:
	// 1. OAuth authorization 요청
	// 2. BMW 로그인 페이지로 리다이렉트
	// 3. 로그인 폼 제출
	// 4. Authorization code 받기
	// 5. Token exchange
	// 이 과정은 JavaScript 실행이 필요할 수 있어 headless browser가 필요할 수 있습니다

	a.isLoggedIn = false
	return fmt.Errorf("OAuth2 로그인은 현재 지원되지 않습니다. Selenium이나 수동 쿠키 설정이 필요합니다")
}

// extractCSRFToken extracts CSRF token from HTML
func (a *AuthClient) extractCSRFToken(html []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return ""
	}

	// Look for CSRF token in various possible locations
	token, exists := doc.Find("input[name='_csrf']").Attr("value")
	if exists {
		return token
	}

	token, exists = doc.Find("input[name='csrf_token']").Attr("value")
	if exists {
		return token
	}

	token, exists = doc.Find("meta[name='csrf-token']").Attr("content")
	if exists {
		return token
	}

	return ""
}

// checkLoginSuccess checks if login was successful
func (a *AuthClient) checkLoginSuccess(resp *http.Response) bool {
	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// Check for session cookies
		cookies := a.client.Jar.Cookies(resp.Request.URL)
		for _, cookie := range cookies {
			// Look for session-related cookies
			if strings.Contains(strings.ToLower(cookie.Name), "session") ||
				strings.Contains(strings.ToLower(cookie.Name), "auth") ||
				strings.Contains(strings.ToLower(cookie.Name), "jsessionid") {
				return true
			}
		}

		// Check redirect location
		location := resp.Header.Get("Location")
		if location != "" && !strings.Contains(location, "login") {
			return true
		}
	}

	return false
}

// Get performs an authenticated GET request
func (a *AuthClient) Get(url string) (*http.Response, error) {
	if !a.isLoggedIn {
		if err := a.Login(); err != nil {
			return nil, err
		}
	}

	resp, err := a.client.Get(url)
	if err != nil {
		return nil, err
	}

	// Check if we need to re-login
	if a.needsRelogin(resp) {
		resp.Body.Close()
		a.isLoggedIn = false
		if err := a.Login(); err != nil {
			return nil, err
		}
		return a.client.Get(url)
	}

	return resp, nil
}

// needsRelogin checks if we need to login again
func (a *AuthClient) needsRelogin(resp *http.Response) bool {
	// Check if redirected to login page
	if resp.Request.URL.Path == "/login" || strings.Contains(resp.Request.URL.Path, "login") {
		return true
	}

	// Check status code
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return true
	}

	return false
}

// IsLoggedIn returns login status
func (a *AuthClient) IsLoggedIn() bool {
	return a.isLoggedIn
}
