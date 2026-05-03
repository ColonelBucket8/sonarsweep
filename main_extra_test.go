package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_StripsTrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")
	configPath = tmpFile
	defer func() { configPath = "" }()

	jsonContent := `{
		"sonarqube_url": "http://example.com:9000/"
	}`
	if err := os.WriteFile(tmpFile, []byte(jsonContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg := loadConfig()

	if cfg.SonarURL != "http://example.com:9000" {
		t.Errorf("Expected trailing slash to be stripped, got %s", cfg.SonarURL)
	}
}

func TestExportToCSV_CreatesFileAndHandlesErrors(t *testing.T) {
	tmpDir := t.TempDir()
	tmpExportFile := filepath.Join(tmpDir, "custom_export.csv")
	
	// Temporarily override cliExportPath
	cliExportPath = tmpExportFile
	defer func() { cliExportPath = "" }()

	issues := []Issue{
		{
			Key:       "ISSUE-1",
			Rule:      "rule-1",
			Severity:  "HIGH",
			Component: "project:src/main.go",
			Line:      10,
			Message:   "Bad code",
			Status:    "OPEN",
			Effort:    "10min",
		},
	}

	savedFile, err := exportToCSV(issues, "project")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if savedFile != tmpExportFile {
		t.Errorf("Expected savedFile to be %s, got %s", tmpExportFile, savedFile)
	}

	// Verify file was actually created and contains data
	file, err := os.Open(tmpExportFile)
	if err != nil {
		t.Fatalf("Expected exported file to exist, got error: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 rows (header + 1 issue), got %d", len(records))
	}

	if records[1][0] != "ISSUE-1" {
		t.Errorf("Expected first column of data to be 'ISSUE-1', got %s", records[1][0])
	}
}
