package scraper

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseReservationPage parses the reservation page HTML
func ParseReservationPage(html []byte) (map[string]bool, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패 (failed to parse HTML): %w", err)
	}

	availablePrograms := make(map[string]bool)

	// Look for program cards/items
	// This selector will need to be adjusted based on actual HTML structure
	doc.Find(".program-item, .course-item, .product-item").Each(func(i int, s *goquery.Selection) {
		// Extract program name
		programName := strings.TrimSpace(s.Find(".title, .name, h3, h4").Text())
		
		// Check if available (not sold out)
		statusText := strings.ToLower(s.Find(".status, .availability").Text())
		buttonText := strings.ToLower(s.Find("button, .btn").Text())
		
		isAvailable := true
		if strings.Contains(statusText, "매진") || 
		   strings.Contains(statusText, "마감") ||
		   strings.Contains(statusText, "sold out") ||
		   strings.Contains(buttonText, "매진") ||
		   strings.Contains(buttonText, "마감") {
			isAvailable = false
		}
		
		if programName != "" {
			availablePrograms[programName] = isAvailable
		}
	})

	return availablePrograms, nil
}

// ParseProgramListPage parses the program list page to extract all available programs
func ParseProgramListPage(html []byte) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패 (failed to parse HTML): %w", err)
	}

	var programs []string

	// Extract program names from the list
	// Adjust selectors based on actual HTML structure
	doc.Find(".program-name, .course-name, td.name, .list-item").Each(func(i int, s *goquery.Selection) {
		programName := strings.TrimSpace(s.Text())
		if programName != "" && !strings.Contains(strings.ToLower(programName), "total") {
			programs = append(programs, programName)
		}
	})

	// Also check for table structures
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		// Skip header rows
		if i == 0 {
			return
		}
		
		// Get first column which typically contains program name
		programName := strings.TrimSpace(s.Find("td:first-child").Text())
		if programName != "" {
			programs = append(programs, programName)
		}
	})

	return programs, nil
}