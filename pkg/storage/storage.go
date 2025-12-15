package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
)

type Storage struct {
	config *config.StorageConfig
	log    *logger.Logger
	mu     sync.RWMutex
}

type Connection struct {
	ProfileURL     string    `json:"profile_url"`
	Name           string    `json:"name"`
	Title          string    `json:"title,omitempty"`
	Company        string    `json:"company,omitempty"`
	Location       string    `json:"location,omitempty"`
	RequestedAt    time.Time `json:"requested_at"`
	AcceptedAt     time.Time `json:"accepted_at,omitempty"`
	Status         string    `json:"status"`
	Note           string    `json:"note,omitempty"`
	MessageSent    bool      `json:"message_sent"`
	MessageSentAt  time.Time `json:"message_sent_at,omitempty"`
}

type Message struct {
	ProfileURL   string    `json:"profile_url"`
	RecipientName string   `json:"recipient_name"`
	Content      string    `json:"content"`
	Template     string    `json:"template,omitempty"`
	SentAt       time.Time `json:"sent_at"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
}

type Profile struct {
	URL            string    `json:"url"`
	Name           string    `json:"name"`
	Title          string    `json:"title,omitempty"`
	Company        string    `json:"company,omitempty"`
	Location       string    `json:"location,omitempty"`
	Industry       string    `json:"industry,omitempty"`
	ConnectionDegree string  `json:"connection_degree,omitempty"`
	FoundAt        time.Time `json:"found_at"`
	Source         string    `json:"source,omitempty"`
	Processed      bool      `json:"processed"`
}

type Session struct {
	Cookies       []CookieData `json:"cookies"`
	LastLogin     time.Time    `json:"last_login"`
	Email         string       `json:"email"`
	ProfileURL    string       `json:"profile_url,omitempty"`
	IsValid       bool         `json:"is_valid"`
}

type CookieData struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	HTTPOnly bool      `json:"http_only"`
	Secure   bool      `json:"secure"`
}

type DailyStats struct {
	Date               string `json:"date"`
	ConnectionsSent    int    `json:"connections_sent"`
	ConnectionsAccepted int   `json:"connections_accepted"`
	MessagesSent       int    `json:"messages_sent"`
	SearchesPerformed  int    `json:"searches_performed"`
	ProfilesViewed     int    `json:"profiles_viewed"`
}

func New(cfg *config.StorageConfig) (*Storage, error) {
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Storage{
		config: cfg,
		log:    logger.WithComponent("storage"),
	}, nil
}

func (s *Storage) filepath(filename string) string {
	return filepath.Join(s.config.DataDir, filename)
}

func (s *Storage) load(filename string, v interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filepath(filename))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	return nil
}

func (s *Storage) save(filename string, v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(s.filepath(filename), data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

func (s *Storage) LoadConnections() ([]Connection, error) {
	var connections []Connection
	if err := s.load(s.config.ConnectionsFile, &connections); err != nil {
		return nil, err
	}
	if connections == nil {
		connections = []Connection{}
	}
	return connections, nil
}

func (s *Storage) SaveConnections(connections []Connection) error {
	return s.save(s.config.ConnectionsFile, connections)
}

func (s *Storage) AddConnection(conn Connection) error {
	connections, err := s.LoadConnections()
	if err != nil {
		return err
	}

	for i, existing := range connections {
		if existing.ProfileURL == conn.ProfileURL {
			connections[i] = conn
			return s.SaveConnections(connections)
		}
	}

	connections = append(connections, conn)
	return s.SaveConnections(connections)
}

func (s *Storage) GetConnection(profileURL string) (*Connection, error) {
	connections, err := s.LoadConnections()
	if err != nil {
		return nil, err
	}

	for _, conn := range connections {
		if conn.ProfileURL == profileURL {
			return &conn, nil
		}
	}

	return nil, nil
}

func (s *Storage) LoadMessages() ([]Message, error) {
	var messages []Message
	if err := s.load(s.config.MessagesFile, &messages); err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []Message{}
	}
	return messages, nil
}

func (s *Storage) SaveMessages(messages []Message) error {
	return s.save(s.config.MessagesFile, messages)
}

func (s *Storage) AddMessage(msg Message) error {
	messages, err := s.LoadMessages()
	if err != nil {
		return err
	}

	messages = append(messages, msg)
	return s.SaveMessages(messages)
}

func (s *Storage) LoadProfiles() ([]Profile, error) {
	var profiles []Profile
	if err := s.load(s.config.ProfilesFile, &profiles); err != nil {
		return nil, err
	}
	if profiles == nil {
		profiles = []Profile{}
	}
	return profiles, nil
}

func (s *Storage) SaveProfiles(profiles []Profile) error {
	return s.save(s.config.ProfilesFile, profiles)
}

func (s *Storage) AddProfile(profile Profile) error {
	profiles, err := s.LoadProfiles()
	if err != nil {
		return err
	}

	for _, existing := range profiles {
		if existing.URL == profile.URL {
			return nil
		}
	}

	profiles = append(profiles, profile)
	return s.SaveProfiles(profiles)
}

func (s *Storage) GetUnprocessedProfiles(limit int) ([]Profile, error) {
	profiles, err := s.LoadProfiles()
	if err != nil {
		return nil, err
	}

	var unprocessed []Profile
	for _, p := range profiles {
		if !p.Processed {
			unprocessed = append(unprocessed, p)
			if len(unprocessed) >= limit {
				break
			}
		}
	}

	return unprocessed, nil
}

func (s *Storage) MarkProfileProcessed(profileURL string) error {
	profiles, err := s.LoadProfiles()
	if err != nil {
		return err
	}

	for i, p := range profiles {
		if p.URL == profileURL {
			profiles[i].Processed = true
			return s.SaveProfiles(profiles)
		}
	}

	return nil
}

func (s *Storage) LoadSession() (*Session, error) {
	var session Session
	if err := s.load(s.config.SessionFile, &session); err != nil {
		return nil, err
	}

	if session.LastLogin.IsZero() {
		return nil, nil
	}

	return &session, nil
}

func (s *Storage) SaveSession(session *Session) error {
	return s.save(s.config.SessionFile, session)
}

func (s *Storage) GetTodayStats() (*DailyStats, error) {
	today := time.Now().Format("2006-01-02")
	
	var allStats []DailyStats
	if err := s.load("stats.json", &allStats); err != nil {
		return nil, err
	}

	for _, stat := range allStats {
		if stat.Date == today {
			return &stat, nil
		}
	}

	return &DailyStats{Date: today}, nil
}

func (s *Storage) UpdateTodayStats(update func(*DailyStats)) error {
	today := time.Now().Format("2006-01-02")
	
	var allStats []DailyStats
	if err := s.load("stats.json", &allStats); err != nil {
		return err
	}

	found := false
	for i, stat := range allStats {
		if stat.Date == today {
			update(&allStats[i])
			found = true
			break
		}
	}

	if !found {
		newStats := DailyStats{Date: today}
		update(&newStats)
		allStats = append(allStats, newStats)
	}

	return s.save("stats.json", allStats)
}

func (s *Storage) GetConnectionCount(since time.Time) (int, error) {
	connections, err := s.LoadConnections()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, conn := range connections {
		if conn.RequestedAt.After(since) {
			count++
		}
	}

	return count, nil
}

func (s *Storage) GetMessageCount(since time.Time) (int, error) {
	messages, err := s.LoadMessages()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, msg := range messages {
		if msg.SentAt.After(since) {
			count++
		}
	}

	return count, nil
}

func (s *Storage) ProfileExists(profileURL string) (bool, error) {
	profiles, err := s.LoadProfiles()
	if err != nil {
		return false, err
	}

	for _, p := range profiles {
		if p.URL == profileURL {
			return true, nil
		}
	}

	return false, nil
}

func (s *Storage) ConnectionExists(profileURL string) (bool, error) {
	conn, err := s.GetConnection(profileURL)
	if err != nil {
		return false, err
	}
	return conn != nil, nil
}
