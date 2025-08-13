package scraper

import (
	"bmw-driving-center-alter/internal/models"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Scraper handles web scraping operations
type Scraper struct {
	client         *http.Client
	reservationURL string
	programListURL string
}

// New creates a new Scraper instance
func New(reservationURL, programListURL string) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		reservationURL: reservationURL,
		programListURL: programListURL,
	}
}

// CheckReservationStatus checks if programs are available for reservation
func (s *Scraper) CheckReservationStatus(programs []models.Program) (*models.ReservationStatus, error) {
	// Fetch the reservation page
	resp, err := s.client.Get(s.reservationURL)
	if err != nil {
		return nil, fmt.Errorf("예약 페이지 요청 실패 (failed to fetch reservation page): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패 (failed to read response): %w", err)
	}

	content := string(body)
	status := &models.ReservationStatus{
		Programs:  make([]models.Program, len(programs)),
		CheckedAt: time.Now(),
	}

	// Check each program
	for i, program := range programs {
		status.Programs[i] = program
		status.Programs[i].LastChecked = time.Now()
		
		// Check if any keyword matches and reservation is available
		for _, keyword := range program.Keywords {
			if strings.Contains(content, keyword) {
				// Check if the program is available (not sold out)
				// This logic will need to be adjusted based on actual HTML structure
				if !strings.Contains(content, keyword + ".*매진") && 
				   !strings.Contains(content, keyword + ".*마감") {
					status.Programs[i].IsOpen = true
					status.HasOpenings = true
					break
				}
			}
		}
	}

	return status, nil
}

// FetchProgramList fetches available programs from the program list page
func (s *Scraper) FetchProgramList() ([]string, error) {
	resp, err := s.client.Get(s.programListURL)
	if err != nil {
		return nil, fmt.Errorf("프로그램 목록 페이지 요청 실패 (failed to fetch program list): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패 (failed to read response): %w", err)
	}

	// Parse the HTML to extract program names
	// This is a simplified version - actual implementation will need proper HTML parsing
	content := string(body)
	var programs []string
	
	// TODO: Implement actual HTML parsing logic
	// For now, return a placeholder
	_ = content
	
	return programs, nil
}