package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	// Save original configPath and restore after test
	originalConfigPath := configPath
	configPath = "nonexistent_config_file_12345.json"
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.SonarURL != "" {
		t.Errorf("Expected empty SonarURL for non-existent file, got %s", cfg.SonarURL)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("Expected empty Projects for non-existent file, got %d", len(cfg.Projects))
	}
	if len(cfg.SoftwareQualities) != 3 {
		t.Errorf("Expected 3 default SoftwareQualities, got %d", len(cfg.SoftwareQualities))
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	testConfig := Config{
		SonarURL:          "http://test.example.com:9000",
		Projects:          []string{"test-project-1", "test-project-2"},
		SoftwareQualities: []string{"RELIABILITY", "SECURITY"},
	}

	data, _ := json.Marshal(testConfig)
	os.WriteFile(tmpFile, data, 0644)

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.SonarURL != testConfig.SonarURL {
		t.Errorf("Expected SonarURL %s, got %s", testConfig.SonarURL, cfg.SonarURL)
	}
	if len(cfg.Projects) != len(testConfig.Projects) {
		t.Errorf("Expected %d projects, got %d", len(testConfig.Projects), len(cfg.Projects))
	}
	if cfg.Projects[0] != "test-project-1" || cfg.Projects[1] != "test-project-2" {
		t.Errorf("Projects not parsed correctly, got %v", cfg.Projects)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	os.WriteFile(tmpFile, []byte("this is not valid json{"), 0644)

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.SonarURL != "" || len(cfg.Projects) != 0 {
		t.Errorf("Expected default config for invalid JSON, got %+v", cfg)
	}
}

func TestSaveConfig_AndLoadConfig_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	original := Config{
		SonarURL:          "http://roundtrip.test:9000",
		Projects:          []string{"project-a", "project-b", "project-c"},
		SoftwareQualities: []string{"MAINTAINABILITY"},
	}

	saveConfig(original)

	loaded := loadConfig()

	if loaded.SonarURL != original.SonarURL {
		t.Errorf("SonarURL mismatch after round-trip: got %s, want %s", loaded.SonarURL, original.SonarURL)
	}
	if len(loaded.Projects) != len(original.Projects) {
		t.Errorf("Projects count mismatch after round-trip: got %d, want %d", len(loaded.Projects), len(original.Projects))
	}
	if loaded.Projects[0] != "project-a" || loaded.Projects[2] != "project-c" {
		t.Errorf("Projects content mismatch after round-trip: got %v", loaded.Projects)
	}
	if loaded.SoftwareQualities[0] != "MAINTAINABILITY" {
		t.Errorf("SoftwareQualities mismatch after round-trip: got %v", loaded.SoftwareQualities)
	}
}

func TestSaveConfig_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "new_config.json")

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := Config{
		SonarURL:          "http://createtest.com:9000",
		Projects:          []string{"auto-created-project"},
		SoftwareQualities: []string{"RELIABILITY", "SECURITY", "MAINTAINABILITY"},
	}

	saveConfig(cfg)

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("Expected config file to be created at %s", tmpFile)
	}
}

func TestDefaultConfig_HasCorrectSoftwareQualities(t *testing.T) {
	if len(defaultConfig.SoftwareQualities) != 3 {
		t.Errorf("Expected 3 default software qualities, got %d", len(defaultConfig.SoftwareQualities))
	}

	expected := []string{"RELIABILITY", "SECURITY", "MAINTAINABILITY"}
	for i, sq := range expected {
		if defaultConfig.SoftwareQualities[i] != sq {
			t.Errorf("Expected SoftwareQuality[%d] to be %s, got %s", i, sq, defaultConfig.SoftwareQualities[i])
		}
	}
}

func TestConfig_WithToken(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	testConfig := Config{
		SonarURL:          "http://test.example.com:9000",
		Projects:          []string{"test-project"},
		SoftwareQualities: []string{"RELIABILITY", "SECURITY"},
		Token:             "squ_test_token_12345",
	}

	data, _ := json.Marshal(testConfig)
	os.WriteFile(tmpFile, data, 0644)

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.Token != testConfig.Token {
		t.Errorf("Expected Token %s, got %s", testConfig.Token, cfg.Token)
	}
}

func TestSaveConfig_AndLoadConfig_RoundTripWithToken(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	original := Config{
		SonarURL:          "http://roundtrip.test:9000",
		Projects:          []string{"project-a"},
		SoftwareQualities: []string{"MAINTAINABILITY"},
		Token:             "squ_secret_token_abc123",
	}

	saveConfig(original)

	loaded := loadConfig()

	if loaded.Token != original.Token {
		t.Errorf("Token mismatch after round-trip: got %s, want %s", loaded.Token, original.Token)
	}
}

func TestConfig_EmptyTokenOmitsFromJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	testConfig := Config{
		SonarURL:          "http://test.example.com:9000",
		Projects:          []string{"test-project"},
		SoftwareQualities: []string{"RELIABILITY"},
		Token:             "",
	}

	data, _ := json.Marshal(testConfig)
	os.WriteFile(tmpFile, data, 0644)

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.Token != "" {
		t.Errorf("Expected empty Token, got %s", cfg.Token)
	}
}

func TestIssue_SeverityMapping(t *testing.T) {
	issue := Issue{
		Key:      "test-key-123",
		Rule:     "java:S1234",
		Severity: "HIGH",
		Component: "test-project:src/Main.java",
		Line:     42,
		Message:  "Test issue message",
		Status:   "OPEN",
	}

	if issue.Severity != "HIGH" {
		t.Errorf("Expected Severity HIGH, got %s", issue.Severity)
	}
	if issue.Line != 42 {
		t.Errorf("Expected Line 42, got %d", issue.Line)
	}
}

func TestIssue_Impacts(t *testing.T) {
	issue := Issue{
		Key:      "test-key-456",
		Rule:     "java:S5678",
		Severity: "MEDIUM",
		Component: "test-project:src/Test.java",
		Impacts: []struct {
			SoftwareQuality string `json:"softwareQuality"`
			Severity        string `json:"severity"`
		}{
			{SoftwareQuality: "MAINTAINABILITY", Severity: "MEDIUM"},
			{SoftwareQuality: "RELIABILITY", Severity: "LOW"},
		},
	}

	if len(issue.Impacts) != 2 {
		t.Errorf("Expected 2 impacts, got %d", len(issue.Impacts))
	}

	if issue.Impacts[0].SoftwareQuality != "MAINTAINABILITY" {
		t.Errorf("Expected first impact MAINTAINABILITY, got %s", issue.Impacts[0].SoftwareQuality)
	}
}
