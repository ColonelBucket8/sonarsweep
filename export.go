package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var cliExportPath string

func exportToCSV(issues []Issue, projectKey string) (string, error) {
	dateStr := time.Now().Format("20060102_150405")

	var outputFilename string
	if cliExportPath != "" {
		outputFilename = cliExportPath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}

		downloadsDir := filepath.Join(homeDir, "Downloads")
		projectDir := filepath.Join(downloadsDir, projectKey)

		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		outputFilename = filepath.Join(projectDir, fmt.Sprintf("sonarqube_issues_%s.csv", dateStr))
	}

	file, err := os.Create(outputFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	fields := []string{
		"key", "rule", "impact_severity", "component", "line",
		"message", "status", "effort", "author", "creationDate",
	}
	writer.Write(fields)

	prefix := projectKey + ":"

	for _, issue := range issues {
		component := issue.Component
		if strings.HasPrefix(component, prefix) {
			component = strings.TrimPrefix(component, prefix)
		}

		lineStr := ""
		if issue.Line != 0 {
			lineStr = strconv.Itoa(issue.Line)
		}

		row := []string{
			issue.Key,
			issue.Rule,
			issue.Severity,
			component,
			lineStr,
			issue.Message,
			issue.Status,
			issue.Effort,
			issue.Author,
			issue.CreationDate,
		}
		writer.Write(row)
	}

	return outputFilename, nil
}
