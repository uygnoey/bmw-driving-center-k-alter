package notifier

import (
	"bmw-driving-center-alter/internal/config"
	"bmw-driving-center-alter/internal/models"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// EmailNotifier handles email notifications
type EmailNotifier struct {
	config config.EmailConfig
	auth   smtp.Auth
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(cfg config.EmailConfig) *EmailNotifier {
	auth := smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Host)
	
	return &EmailNotifier{
		config: cfg,
		auth:   auth,
	}
}

// SendNotification sends an email notification about available programs
func (e *EmailNotifier) SendNotification(status *models.ReservationStatus) error {
	// Filter only open programs
	var openPrograms []models.Program
	for _, program := range status.Programs {
		if program.IsOpen {
			openPrograms = append(openPrograms, program)
		}
	}

	if len(openPrograms) == 0 {
		return nil // No open programs to notify about
	}

	// Build email body
	body := e.buildEmailBody(openPrograms, status.CheckedAt)
	
	// Build the email message
	message := e.buildMessage(body)

	// Send to all recipients
	addr := fmt.Sprintf("%s:%d", e.config.SMTP.Host, e.config.SMTP.Port)
	err := smtp.SendMail(addr, e.auth, e.config.From, e.config.To, []byte(message))
	
	if err != nil {
		return fmt.Errorf("이메일 전송 실패 (failed to send email): %w", err)
	}

	return nil
}

// buildEmailBody creates the email body content
func (e *EmailNotifier) buildEmailBody(programs []models.Program, checkedAt time.Time) string {
	var sb strings.Builder
	
	sb.WriteString("BMW 드라이빙 센터 예약이 오픈되었습니다!\n")
	sb.WriteString("BMW Driving Center reservations are now open!\n\n")
	
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	
	sb.WriteString("🚗 예약 가능한 프로그램 (Available Programs):\n\n")
	
	for _, program := range programs {
		sb.WriteString(fmt.Sprintf("  ✅ %s\n", program.Name))
	}
	
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	
	sb.WriteString("📅 예약 페이지 (Reservation Page):\n")
	sb.WriteString("   https://driving-center.bmw.co.kr/orders/programs/products/view\n\n")
	
	sb.WriteString(fmt.Sprintf("🕐 확인 시간 (Checked at): %s\n", 
		checkedAt.Format("2006-01-02 15:04:05")))
	
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("⚡ 빠른 예약을 권장합니다! (Book quickly before it fills up!)\n")
	
	return sb.String()
}

// buildMessage creates the full email message with headers
func (e *EmailNotifier) buildMessage(body string) string {
	headers := make(map[string]string)
	headers["From"] = e.config.From
	headers["To"] = strings.Join(e.config.To, ", ")
	headers["Subject"] = e.config.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"
	
	var message strings.Builder
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(body)
	
	return message.String()
}

// TestConnection tests the email configuration
func (e *EmailNotifier) TestConnection() error {
	testBody := "BMW 드라이빙 센터 모니터 테스트 이메일입니다.\n"
	testBody += "This is a test email from BMW Driving Center Monitor.\n"
	testBody += fmt.Sprintf("Time: %s", time.Now().Format("2006-01-02 15:04:05"))
	
	message := e.buildMessage(testBody)
	addr := fmt.Sprintf("%s:%d", e.config.SMTP.Host, e.config.SMTP.Port)
	
	return smtp.SendMail(addr, e.auth, e.config.From, e.config.To, []byte(message))
}