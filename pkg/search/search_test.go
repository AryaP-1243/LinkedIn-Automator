package search

import (
	"testing"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/storage"
)

func TestDuplicateProfileDetection(t *testing.T) {
	// Create a mock storage
	cfg := &config.StorageConfig{
		DataDir:         t.TempDir(),
		ConnectionsFile: "connections.json",
		MessagesFile:    "messages.json",
		SessionFile:     "session.json",
		ProfilesFile:    "profiles.json",
	}

	store, err := storage.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create searcher with nil browser (we're only testing filtering logic)
	searcher := &Searcher{
		storage: store,
	}

	t.Run("Filter duplicates within results", func(t *testing.T) {
		results := []SearchResult{
			{ProfileURL: "https://www.linkedin.com/in/user1/", Name: "User 1"},
			{ProfileURL: "https://www.linkedin.com/in/user2/", Name: "User 2"},
			{ProfileURL: "https://www.linkedin.com/in/user1/", Name: "User 1 Duplicate"},
			{ProfileURL: "https://www.linkedin.com/in/user3/", Name: "User 3"},
		}

		unique := searcher.filterDuplicates(results)

		if len(unique) != 3 {
			t.Errorf("Expected 3 unique results, got %d", len(unique))
		}

		// Verify user1 appears only once
		user1Count := 0
		for _, r := range unique {
			if r.ProfileURL == "https://www.linkedin.com/in/user1/" {
				user1Count++
			}
		}

		if user1Count != 1 {
			t.Errorf("Expected user1 to appear once, appeared %d times", user1Count)
		}
	})

	t.Run("Filter duplicates from storage", func(t *testing.T) {
		// Add a profile to storage
		existingProfile := storage.Profile{
			URL:  "https://www.linkedin.com/in/existing/",
			Name: "Existing User",
		}
		if err := store.AddProfile(existingProfile); err != nil {
			t.Fatalf("Failed to add profile: %v", err)
		}

		results := []SearchResult{
			{ProfileURL: "https://www.linkedin.com/in/new/", Name: "New User"},
			{ProfileURL: "https://www.linkedin.com/in/existing/", Name: "Existing User"},
		}

		unique := searcher.filterDuplicates(results)

		// Should only return the new profile
		if len(unique) != 1 {
			t.Errorf("Expected 1 unique result, got %d", len(unique))
		}

		if unique[0].ProfileURL != "https://www.linkedin.com/in/new/" {
			t.Errorf("Expected new profile, got %s", unique[0].ProfileURL)
		}
	})

	t.Run("Filter empty URLs", func(t *testing.T) {
		results := []SearchResult{
			{ProfileURL: "", Name: "No URL"},
			{ProfileURL: "https://www.linkedin.com/in/user1/", Name: "User 1"},
		}

		unique := searcher.filterDuplicates(results)

		if len(unique) != 1 {
			t.Errorf("Expected 1 unique result, got %d", len(unique))
		}
	})
}

func TestProfileURLNormalization(t *testing.T) {
	searcher := &Searcher{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean URL",
			input:    "https://www.linkedin.com/in/johndoe/",
			expected: "https://www.linkedin.com/in/johndoe/",
		},
		{
			name:     "URL with query params",
			input:    "https://www.linkedin.com/in/johndoe/?trk=search",
			expected: "https://www.linkedin.com/in/johndoe/",
		},
		{
			name:     "URL without trailing slash",
			input:    "https://www.linkedin.com/in/johndoe",
			expected: "https://www.linkedin.com/in/johndoe/",
		},
		{
			name:     "Relative URL",
			input:    "/in/johndoe/",
			expected: "https://www.linkedin.com/in/johndoe/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := searcher.normalizeProfileURL(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSearchQueryBuilder(t *testing.T) {
	searcher := &Searcher{}

	tests := []struct {
		name     string
		query    SearchQuery
		contains []string
	}{
		{
			name: "Keywords only",
			query: SearchQuery{
				Keywords: "software engineer",
			},
			contains: []string{"keywords=software+engineer"},
		},
		{
			name: "Job title",
			query: SearchQuery{
				JobTitle: "Senior Developer",
			},
			contains: []string{"title=Senior+Developer"},
		},
		{
			name: "Company",
			query: SearchQuery{
				Company: "Google",
			},
			contains: []string{"company=Google"},
		},
		{
			name: "Multiple filters",
			query: SearchQuery{
				Keywords: "AI",
				Company:  "OpenAI",
			},
			contains: []string{"keywords=AI", "company=OpenAI"},
		},
		{
			name: "With pagination",
			query: SearchQuery{
				Keywords: "test",
				Page:     2,
			},
			contains: []string{"keywords=test", "page=2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := searcher.buildSearchURL(tt.query)

			for _, expected := range tt.contains {
				if !contains(url, expected) {
					t.Errorf("URL %s should contain %s", url, expected)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
