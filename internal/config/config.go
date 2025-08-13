package config

import (
	"bmw-driving-center-alter/internal/models"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Auth     AuthConfig          `yaml:"auth"`
	Monitor  MonitorConfig       `yaml:"monitor"`
	Programs []models.Program    `yaml:"programs"`
	Email    EmailConfig         `yaml:"email"`
}

// AuthConfig represents authentication settings
type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MonitorConfig represents monitoring settings
type MonitorConfig struct {
	Interval        int    `yaml:"interval"`          // in seconds
	ReservationURL  string `yaml:"reservation_url"`   // 예약 페이지 URL
	ProgramListURL  string `yaml:"program_list_url"`  // 프로그램 목록 URL
}

// EmailConfig represents email notification settings
type EmailConfig struct {
	SMTP SMTPConfig `yaml:"smtp"`
	From string     `yaml:"from"`
	To   []string   `yaml:"to"`
	Subject string  `yaml:"subject"`
}

// SMTPConfig represents SMTP server settings
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// GetConfigPath finds the configuration file path
func GetConfigPath() string {
	// 1. 실행 파일과 같은 디렉토리의 configs/config.yaml
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	
	// .app 번들 내부에서 실행되는 경우 Resources 디렉토리 확인
	if filepath.Base(filepath.Dir(execDir)) == "MacOS" {
		// MacOS 디렉토리의 상위인 Contents 디렉토리
		contentsDir := filepath.Dir(execDir)
		resourcesPath := filepath.Join(contentsDir, "Resources", "configs", "config.yaml")
		if _, err := os.Stat(resourcesPath); err == nil {
			return resourcesPath
		}
	}
	
	// 일반 실행 파일 경로
	configPath := filepath.Join(execDir, "configs", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	
	// 2. 현재 디렉토리의 configs/config.yaml
	configPath = filepath.Join("configs", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	
	// 3. 홈 디렉토리의 .bmw-driving-center/config.yaml
	homeDir, _ := os.UserHomeDir()
	configPath = filepath.Join(homeDir, ".bmw-driving-center", "config.yaml")
	
	// 디렉토리가 없으면 생성
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}
	
	return configPath
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// 경로가 비어있으면 자동 탐색
	if path == "" {
		path = GetConfigPath()
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("설정 파일 읽기 실패 (failed to read config file): %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("설정 파일 파싱 실패 (failed to parse config file): %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to file
func Save(path string, cfg *Config) error {
	// 경로가 비어있으면 자동 탐색
	if path == "" {
		path = GetConfigPath()
	}
	
	// 디렉토리 생성
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}
	
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("설정 직렬화 실패: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("설정 파일 저장 실패: %w", err)
	}
	
	return nil
}