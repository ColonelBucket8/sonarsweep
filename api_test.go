package main

import (
	"testing"
)

func TestIssue_SeverityMapping(t *testing.T) {
	issue := Issue{
		Key:       "test-key-123",
		Rule:      "java:S1234",
		Severity:  "HIGH",
		Component: "test-project:src/Main.java",
		Line:      42,
		Message:   "Test issue message",
		Status:    "OPEN",
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
		Key:       "test-key-456",
		Rule:      "java:S5678",
		Severity:  "MEDIUM",
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
