package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LinkedIn    LinkedInConfig    `yaml:"linkedin" json:"linkedin"`
	Browser     BrowserConfig     `yaml:"browser" json:"browser"`
	Stealth     StealthConfig     `yaml:"stealth" json:"stealth"`
	RateLimits  RateLimitConfig   `yaml:"rate_limits" json:"rate_limits"`
	Messaging   MessagingConfig   `yaml:"messaging" json:"messaging"`
	Search      SearchConfig      `yaml:"search" json:"search"`
	Storage     StorageConfig     `yaml:"storage" json:"storage"`
	Logging     LoggingConfig     `yaml:"logging" json:"logging"`
	Schedule    ScheduleConfig    `yaml:"schedule" json:"schedule"`
}

type LinkedInConfig struct {
	Email    string `yaml:"email" json:"email"`
	Password string `yaml:"password" json:"password"`
}

type BrowserConfig struct {
	Headless        bool     `yaml:"headless" json:"headless"`
	UserDataDir     string   `yaml:"user_data_dir" json:"user_data_dir"`
	ViewportWidth   int      `yaml:"viewport_width" json:"viewport_width"`
	ViewportHeight  int      `yaml:"viewport_height" json:"viewport_height"`
	UserAgents      []string `yaml:"user_agents" json:"user_agents"`
	DisableWebRTC   bool     `yaml:"disable_webrtc" json:"disable_webrtc"`
	DisableWebGL    bool     `yaml:"disable_webgl" json:"disable_webgl"`
}

type StealthConfig struct {
	MouseMovement    MouseMovementConfig    `yaml:"mouse_movement" json:"mouse_movement"`
	Timing           TimingConfig           `yaml:"timing" json:"timing"`
	Scrolling        ScrollingConfig        `yaml:"scrolling" json:"scrolling"`
	Typing           TypingConfig           `yaml:"typing" json:"typing"`
	Fingerprinting   FingerprintConfig      `yaml:"fingerprinting" json:"fingerprinting"`
}

type MouseMovementConfig struct {
	Enabled           bool    `yaml:"enabled" json:"enabled"`
	MinSpeed          float64 `yaml:"min_speed" json:"min_speed"`
	MaxSpeed          float64 `yaml:"max_speed" json:"max_speed"`
	OvershootEnabled  bool    `yaml:"overshoot_enabled" json:"overshoot_enabled"`
	MicroMovements    bool    `yaml:"micro_movements" json:"micro_movements"`
	BezierComplexity  int     `yaml:"bezier_complexity" json:"bezier_complexity"`
}

type TimingConfig struct {
	MinActionDelay    time.Duration `yaml:"min_action_delay" json:"min_action_delay"`
	MaxActionDelay    time.Duration `yaml:"max_action_delay" json:"max_action_delay"`
	MinThinkTime      time.Duration `yaml:"min_think_time" json:"min_think_time"`
	MaxThinkTime      time.Duration `yaml:"max_think_time" json:"max_think_time"`
	PageLoadWait      time.Duration `yaml:"page_load_wait" json:"page_load_wait"`
	HumanVariation    float64       `yaml:"human_variation" json:"human_variation"`
}

type ScrollingConfig struct {
	Enabled             bool    `yaml:"enabled" json:"enabled"`
	MinScrollSpeed      int     `yaml:"min_scroll_speed" json:"min_scroll_speed"`
	MaxScrollSpeed      int     `yaml:"max_scroll_speed" json:"max_scroll_speed"`
	ScrollBackChance    float64 `yaml:"scroll_back_chance" json:"scroll_back_chance"`
	PauseChance         float64 `yaml:"pause_chance" json:"pause_chance"`
	SmoothScrolling     bool    `yaml:"smooth_scrolling" json:"smooth_scrolling"`
}

type TypingConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled"`
	MinKeyDelay       time.Duration `yaml:"min_key_delay" json:"min_key_delay"`
	MaxKeyDelay       time.Duration `yaml:"max_key_delay" json:"max_key_delay"`
	TypoChance        float64       `yaml:"typo_chance" json:"typo_chance"`
	CorrectionDelay   time.Duration `yaml:"correction_delay" json:"correction_delay"`
	ThinkPauseChance  float64       `yaml:"think_pause_chance" json:"think_pause_chance"`
}

type FingerprintConfig struct {
	RotateUserAgent     bool `yaml:"rotate_user_agent" json:"rotate_user_agent"`
	RandomizeViewport   bool `yaml:"randomize_viewport" json:"randomize_viewport"`
	DisableAutomation   bool `yaml:"disable_automation" json:"disable_automation"`
	SpoofTimezone       bool `yaml:"spoof_timezone" json:"spoof_timezone"`
	SpoofLanguage       bool `yaml:"spoof_language" json:"spoof_language"`
}

type RateLimitConfig struct {
	DailyConnectionLimit  int           `yaml:"daily_connection_limit" json:"daily_connection_limit"`
	HourlyConnectionLimit int           `yaml:"hourly_connection_limit" json:"hourly_connection_limit"`
	DailyMessageLimit     int           `yaml:"daily_message_limit" json:"daily_message_limit"`
	HourlyMessageLimit    int           `yaml:"hourly_message_limit" json:"hourly_message_limit"`
	DailySearchLimit      int           `yaml:"daily_search_limit" json:"daily_search_limit"`
	CooldownPeriod        time.Duration `yaml:"cooldown_period" json:"cooldown_period"`
	BreakInterval         time.Duration `yaml:"break_interval" json:"break_interval"`
	BreakDuration         time.Duration `yaml:"break_duration" json:"break_duration"`
}

type MessagingConfig struct {
	Templates         []MessageTemplate `yaml:"templates" json:"templates"`
	FollowUpDelay     time.Duration     `yaml:"follow_up_delay" json:"follow_up_delay"`
	MaxMessageLength  int               `yaml:"max_message_length" json:"max_message_length"`
}

type MessageTemplate struct {
	Name     string `yaml:"name" json:"name"`
	Subject  string `yaml:"subject" json:"subject"`
	Body     string `yaml:"body" json:"body"`
	Type     string `yaml:"type" json:"type"`
}

type SearchConfig struct {
	Keywords        []string `yaml:"keywords" json:"keywords"`
	JobTitles       []string `yaml:"job_titles" json:"job_titles"`
	Companies       []string `yaml:"companies" json:"companies"`
	Locations       []string `yaml:"locations" json:"locations"`
	Industries      []string `yaml:"industries" json:"industries"`
	MaxResults      int      `yaml:"max_results" json:"max_results"`
	PagesPerSearch  int      `yaml:"pages_per_search" json:"pages_per_search"`
}

type StorageConfig struct {
	DataDir           string `yaml:"data_dir" json:"data_dir"`
	ConnectionsFile   string `yaml:"connections_file" json:"connections_file"`
	MessagesFile      string `yaml:"messages_file" json:"messages_file"`
	SessionFile       string `yaml:"session_file" json:"session_file"`
	ProfilesFile      string `yaml:"profiles_file" json:"profiles_file"`
}

type LoggingConfig struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"`
	OutputFile string `yaml:"output_file" json:"output_file"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
}

type ScheduleConfig struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	StartHour     int    `yaml:"start_hour" json:"start_hour"`
	EndHour       int    `yaml:"end_hour" json:"end_hour"`
	Timezone      string `yaml:"timezone" json:"timezone"`
	WorkDays      []int  `yaml:"work_days" json:"work_days"`
	RandomBreaks  bool   `yaml:"random_breaks" json:"random_breaks"`
}

func DefaultConfig() *Config {
	return &Config{
		LinkedIn: LinkedInConfig{
			Email:    os.Getenv("LINKEDIN_EMAIL"),
			Password: os.Getenv("LINKEDIN_PASSWORD"),
		},
		Browser: BrowserConfig{
			Headless:       true,
			UserDataDir:    "./data/browser",
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			UserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
			},
			DisableWebRTC: true,
			DisableWebGL:  false,
		},
		Stealth: StealthConfig{
			MouseMovement: MouseMovementConfig{
				Enabled:          true,
				MinSpeed:         0.5,
				MaxSpeed:         2.0,
				OvershootEnabled: true,
				MicroMovements:   true,
				BezierComplexity: 3,
			},
			Timing: TimingConfig{
				MinActionDelay: 500 * time.Millisecond,
				MaxActionDelay: 2000 * time.Millisecond,
				MinThinkTime:   1000 * time.Millisecond,
				MaxThinkTime:   5000 * time.Millisecond,
				PageLoadWait:   3000 * time.Millisecond,
				HumanVariation: 0.3,
			},
			Scrolling: ScrollingConfig{
				Enabled:          true,
				MinScrollSpeed:   50,
				MaxScrollSpeed:   200,
				ScrollBackChance: 0.1,
				PauseChance:      0.15,
				SmoothScrolling:  true,
			},
			Typing: TypingConfig{
				Enabled:          true,
				MinKeyDelay:      50 * time.Millisecond,
				MaxKeyDelay:      150 * time.Millisecond,
				TypoChance:       0.02,
				CorrectionDelay:  300 * time.Millisecond,
				ThinkPauseChance: 0.05,
			},
			Fingerprinting: FingerprintConfig{
				RotateUserAgent:   true,
				RandomizeViewport: true,
				DisableAutomation: true,
				SpoofTimezone:     true,
				SpoofLanguage:     false,
			},
		},
		RateLimits: RateLimitConfig{
			DailyConnectionLimit:  25,
			HourlyConnectionLimit: 5,
			DailyMessageLimit:     50,
			HourlyMessageLimit:    10,
			DailySearchLimit:      100,
			CooldownPeriod:        30 * time.Minute,
			BreakInterval:         45 * time.Minute,
			BreakDuration:         10 * time.Minute,
		},
		Messaging: MessagingConfig{
			Templates: []MessageTemplate{
				{
					Name:    "connection_request",
					Subject: "",
					Body:    "Hi {{.FirstName}}, I noticed we share an interest in {{.Industry}}. I'd love to connect and learn more about your work at {{.Company}}.",
					Type:    "connection",
				},
				{
					Name:    "follow_up",
					Subject: "Great to connect!",
					Body:    "Hi {{.FirstName}}, thanks for connecting! I'm always interested in {{.Industry}} insights. Would love to hear about your experience at {{.Company}}.",
					Type:    "message",
				},
			},
			FollowUpDelay:    24 * time.Hour,
			MaxMessageLength: 300,
		},
		Search: SearchConfig{
			Keywords:       []string{},
			JobTitles:      []string{},
			Companies:      []string{},
			Locations:      []string{},
			Industries:     []string{},
			MaxResults:     100,
			PagesPerSearch: 5,
		},
		Storage: StorageConfig{
			DataDir:         "./data",
			ConnectionsFile: "connections.json",
			MessagesFile:    "messages.json",
			SessionFile:     "session.json",
			ProfilesFile:    "profiles.json",
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "./data/logs/automation.log",
			MaxSize:    10,
			MaxBackups: 5,
		},
		Schedule: ScheduleConfig{
			Enabled:      true,
			StartHour:    9,
			EndHour:      18,
			Timezone:     "America/New_York",
			WorkDays:     []int{1, 2, 3, 4, 5},
			RandomBreaks: true,
		},
	}
}

func Load(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		ext := filepath.Ext(configPath)
		switch ext {
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse YAML config: %w", err)
			}
		case ".json":
			if err := json.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse JSON config: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported config file format: %s", ext)
		}
	}

	config.applyEnvOverrides()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

func (c *Config) applyEnvOverrides() {
	if email := os.Getenv("LINKEDIN_EMAIL"); email != "" {
		c.LinkedIn.Email = email
	}
	if password := os.Getenv("LINKEDIN_PASSWORD"); password != "" {
		c.LinkedIn.Password = password
	}
	if headless := os.Getenv("BROWSER_HEADLESS"); headless == "false" {
		c.Browser.Headless = false
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.Logging.Level = logLevel
	}
}

func (c *Config) Validate() error {
	if c.LinkedIn.Email == "" {
		return fmt.Errorf("LinkedIn email is required")
	}
	if c.LinkedIn.Password == "" {
		return fmt.Errorf("LinkedIn password is required")
	}
	if c.Browser.ViewportWidth < 800 || c.Browser.ViewportHeight < 600 {
		return fmt.Errorf("viewport dimensions too small")
	}
	if c.RateLimits.DailyConnectionLimit < 1 {
		return fmt.Errorf("daily connection limit must be at least 1")
	}
	return nil
}

func (c *Config) Save(path string) error {
	ext := filepath.Ext(path)
	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(c)
	case ".json":
		data, err = json.MarshalIndent(c, "", "  ")
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
