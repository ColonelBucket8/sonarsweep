package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

var VERSION = "dev"

var configPath string
var cliExportPath string

var (
	SONAR_URL                  string
	availableProjects          []list.Item
	availableSoftwareQualities []string
)

type Config struct {
	SonarURL          string   `json:"sonarqube_url"`
	Projects          []string `json:"projects"`
	SoftwareQualities []string `json:"software_qualities"`
	Token             string   `json:"token,omitempty"`
}

var defaultConfig = Config{
	SonarURL: "",
	Projects: []string{},
	SoftwareQualities: []string{
		"RELIABILITY",
		"SECURITY",
		"MAINTAINABILITY",
	},
}

func isValidURL(s string) bool {
	s = strings.TrimRight(s, "/")
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

func getConfigPath() string {
	if configPath != "" {
		return configPath
	}
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "sonarsweep")
		os.MkdirAll(configDir, 0700)
		return filepath.Join(configDir, "config.json")
	}
	return "sonarsweep.json"
}

func loadConfig() Config {
	path := getConfigPath()
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig
		}
		return defaultConfig
	}
	defer file.Close()
	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return defaultConfig
	}
	config.SonarURL = strings.TrimRight(config.SonarURL, "/")
	if len(config.SoftwareQualities) == 0 {
		config.SoftwareQualities = defaultConfig.SoftwareQualities
	}
	return config
}

func saveConfig(config Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func printHelp() {
	helpText := `SonarSweep v%s - SonarQube Issue Exporter

USAGE:
  sonarsweep [flags]

FLAGS:
  -h, --help           Show this help message and exit
  -v, --version        Show version information and exit
  --reset              Reset the saved URL and Projects from config
  --config <path>      Use a different configuration file
  --view-config        Print current configuration and exit
  --list-projects      List all saved projects and exit
  --add-project <key>  Add a project to the configuration
  --export <path>      Override the CSV export path
  --dry-run            Fetch issues but skip CSV export
  -q, --quiet          Run in headless mode (no TUI)

EXAMPLES:
  sonarsweep --help
  sonarsweep --version
  sonarsweep --reset
  sonarsweep --list-projects
  sonarsweep --add-project my-new-project
  sonarsweep --export /tmp/output.csv
  sonarsweep --dry-run
  sonarsweep --quiet

CONFIGURATION:
  Configuration is stored in '~/.config/sonarsweep/config.json' (fallback: 'sonarsweep.json'). You can edit this file
  directly or use the --add-project flag to add projects.

  The token (USER_TOKEN) must be stored in a .env file, saved in 
  the config file, or entered securely in the TUI prompt.

  Default config location: ~/.config/sonarsweep/config.json
  Override with: --config /path/to/config.json
`
	fmt.Printf(helpText, VERSION)
}

// Styles
var (
	appStyle = lipgloss.NewStyle().Margin(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("63")).
			Padding(0, 1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")).
			MarginBottom(1)

	subtextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	checkboxSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	checkboxUnselectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			Bold(true)

	selectedActionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true).
				Background(lipgloss.Color("235")).
				Padding(0, 1).
				MarginRight(1)

	unselectedActionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	dangerActionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				Bold(true).
				Background(lipgloss.Color("235")).
				Padding(0, 1).
				MarginRight(1)
)

type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string {
	switch i {
	case "+ Add New Project":
		return "Add a new project key"
	case "- Remove Project":
		return "Remove an existing project"
	}
	return "Fetch issues for this project"
}

type sessionState int

const (
	stateURL sessionState = iota
	stateNewProject
	stateToken
	stateProject
	stateQualities
	stateCodePeriod
	stateFetching
	stateDone
	statePrompt
	stateRemoveProject
	stateError
)

type Issue struct {
	Key          string `json:"key"`
	Rule         string `json:"rule"`
	Severity     string `json:"severity"`
	Component    string `json:"component"`
	Line         int    `json:"line"`
	Message      string `json:"message"`
	Status       string `json:"status"`
	Effort       string `json:"effort"`
	Author       string `json:"author"`
	CreationDate string `json:"creationDate"`
	Impacts      []struct {
		SoftwareQuality string `json:"softwareQuality"`
		Severity        string `json:"severity"`
	} `json:"impacts"`
}

type Paging struct {
	Total int `json:"total"`
}

type Response struct {
	Issues []Issue `json:"issues"`
	Paging Paging  `json:"paging"`
	Errors []struct {
		Msg string `json:"msg"`
	} `json:"errors"`
}

type issuesFetchedMsg struct {
	issues []Issue
	err    error
}

func fetchIssues(projectKey, token string, softwareQualities []string, isNewCodePeriod bool) ([]Issue, error) {
	var allIssues []Issue
	client := &http.Client{Timeout: 15 * time.Second}

	p := 1
	ps := 500

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/issues/search", SONAR_URL), nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		q.Add("componentKeys", projectKey)
		q.Add("statuses", "OPEN,CONFIRMED")
		q.Add("impactSeverities", "BLOCKER,HIGH,MEDIUM,LOW")
		q.Add("impactSoftwareQualities", strings.Join(softwareQualities, ","))
		if isNewCodePeriod {
			q.Add("inNewCodePeriod", "true")
		}
		q.Add("p", strconv.Itoa(p))
		q.Add("ps", strconv.Itoa(ps))
		req.URL.RawQuery = q.Encode()

		req.SetBasicAuth(token, "")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("network or connection issue: %w", err)
		}
		defer resp.Body.Close()

		var data Response
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if resp.StatusCode != 200 {
			errMsg := resp.Status
			if len(data.Errors) > 0 {
				errMsg = data.Errors[0].Msg
			}
			if resp.StatusCode == 401 {
				errMsg += "\nTip: This might be an invalid token or lack of permissions."
			}
			return nil, fmt.Errorf("failed to fetch issues (Status Code: %d)\nDetails: %s", resp.StatusCode, errMsg)
		}

		allIssues = append(allIssues, data.Issues...)

		if len(data.Issues) == 0 || len(allIssues) >= data.Paging.Total {
			break
		}
		p++
	}

	// Map modern impacts to the severity field, overriding legacy severities
	for i := range allIssues {
		impactSeverity := "LOW" // Fallback
		for _, impact := range allIssues[i].Impacts {
			for _, sq := range softwareQualities {
				if impact.SoftwareQuality == sq {
					impactSeverity = impact.Severity
					goto FoundImpact
				}
			}
		}
	FoundImpact:
		allIssues[i].Severity = impactSeverity
	}

	return allIssues, nil
}

func fetchIssuesCmd(projectKey, token string, softwareQualities []string, isNewCodePeriod bool) tea.Cmd {
	return func() tea.Msg {
		issues, err := fetchIssues(projectKey, token, softwareQualities, isNewCodePeriod)
		if err != nil {
			return issuesFetchedMsg{err: err}
		}
		return issuesFetchedMsg{issues: issues, err: nil}
	}
}

type model struct {
	state sessionState

	// Config
	config Config

	// UI
	width  int
	height int

	// URL
	urlInput textinput.Model
	urlError string

	// New Project
	newProjectInput textinput.Model
	newProjectError string

	// Token
	tokenInput textinput.Model
	token      string

	// Project
	projectList list.Model
	projectKey  string

	// Remove Project
	removeProjectCursor int

	// Qualities
	qualitiesCursor   int
	selectedQualities map[int]struct{}
	softwareQualities []string

	// Code Period
	codePeriodCursor int
	isNewCodePeriod  bool

	// Prompt
	promptCursor int

	// Fetching
	spinner  spinner.Model
	fetchErr error
	issues   []Issue

	// Done
	summaryTable table.Model
	savedFile    string
}

func initialModel() model {
	godotenv.Load()
	token := os.Getenv("USER_TOKEN")

	config := loadConfig()
	SONAR_URL = config.SonarURL
	availableSoftwareQualities = defaultConfig.SoftwareQualities

	if token == "" {
		token = config.Token
	}

	availableProjects = []list.Item{}
	for _, p := range config.Projects {
		availableProjects = append(availableProjects, item(p))
	}
	if len(config.Projects) > 0 {
		availableProjects = append(availableProjects, item("- Remove Project"), item("+ Add New Project"))
	} else {
		availableProjects = append(availableProjects, item("+ Add New Project"))
	}

	state := stateProject
	if config.SonarURL == "" {
		state = stateURL
	} else if len(config.Projects) == 0 {
		state = stateNewProject
	} else if token == "" {
		state = stateToken
	}

	uiURL := textinput.New()
	uiURL.Placeholder = "http://localhost:9000"
	uiURL.Focus()
	uiURL.CharLimit = 256
	uiURL.Width = 50
	uiURL.PromptStyle = highlightStyle
	uiURL.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	uiProj := textinput.New()
	uiProj.Placeholder = "people-web-ppd"
	uiProj.Focus()
	uiProj.CharLimit = 256
	uiProj.Width = 50
	uiProj.PromptStyle = highlightStyle
	uiProj.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter SonarQube Token"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.PromptStyle = highlightStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("212")).BorderLeftForeground(lipgloss.Color("212"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("212")).BorderLeftForeground(lipgloss.Color("212"))

	pl := list.New(availableProjects, delegate, 0, 0)
	pl.Title = "Select a project"
	pl.SetShowTitle(false)
	pl.SetShowStatusBar(false)
	pl.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		state:             state,
		config:            config,
		token:             token,
		tokenInput:        ti,
		urlInput:          uiURL,
		newProjectInput:   uiProj,
		projectList:       pl,
		selectedQualities: make(map[int]struct{}),
		isNewCodePeriod:   true,
		spinner:           s,
	}
}

func (m model) Init() tea.Cmd {
	if m.state == stateToken || m.state == stateURL || m.state == stateNewProject {
		return textinput.Blink
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update list dimensions accounting for margins and titles
		h, v := appStyle.GetFrameSize()
		m.projectList.SetSize(msg.Width-h, msg.Height-v-12) // -12 to account for the larger ASCII art header

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.state != stateDone && m.state != stateError {
				return m, tea.Quit
			}
		}

		switch m.state {
		case stateURL:
			if msg.Type == tea.KeyEnter {
				url := strings.TrimSpace(m.urlInput.Value())
				url = strings.TrimRight(url, "/")
				if !isValidURL(url) {
					m.urlError = "Invalid URL. Must start with http:// or https:// and have a valid host"
					return m, nil
				}
				m.urlError = ""
				SONAR_URL = url
				m.config.SonarURL = url
				if err := saveConfig(m.config); err != nil {
					m.fetchErr = fmt.Errorf("failed to save config: %w", err)
					m.state = stateError
					return m, nil
				}

				if len(m.config.Projects) == 0 {
					m.state = stateNewProject
				} else if m.token == "" {
					m.state = stateToken
				} else {
					m.state = stateProject
				}

				return m, nil
			}

		case stateNewProject:
			if msg.Type == tea.KeyEnter {
				proj := m.newProjectInput.Value()
				proj = strings.TrimSpace(proj)
				if proj != "" {
					for _, existingProject := range m.config.Projects {
						if existingProject == proj {
							m.newProjectError = "Project already exists!"
							return m, nil
						}
					}
					m.newProjectError = ""
					m.config.Projects = append(m.config.Projects, proj)
					if err := saveConfig(m.config); err != nil {
					m.fetchErr = fmt.Errorf("failed to save config: %w", err)
					m.state = stateError
					return m, nil
				}

					last := len(availableProjects) - 1
					if last >= 0 {
						availableProjects = availableProjects[:last]
					}
					availableProjects = append(availableProjects, item(proj), item("- Remove Project"), item("+ Add New Project"))
					m.projectList.SetItems(availableProjects)

					if m.token == "" {
						m.state = stateToken
					} else {
						m.state = stateProject
					}
				}
				return m, nil
			}

		case stateToken:
			if msg.Type == tea.KeyEnter {
				m.token = m.tokenInput.Value()
				if strings.TrimSpace(m.token) != "" {
					m.config.Token = m.token
					if err := saveConfig(m.config); err != nil {
					m.fetchErr = fmt.Errorf("failed to save config: %w", err)
					m.state = stateError
					return m, nil
				}
					m.state = stateProject
				}
				return m, nil
			}

		case stateProject:
			if msg.Type == tea.KeyEnter {
				if i, ok := m.projectList.SelectedItem().(item); ok {
					if i == "+ Add New Project" {
						m.state = stateNewProject
						return m, nil
					}
					if i == "- Remove Project" {
						m.removeProjectCursor = 0
						m.state = stateRemoveProject
						return m, nil
					}
					m.projectKey = string(i)

					for i, sq := range availableSoftwareQualities {
						for _, configSQ := range m.config.SoftwareQualities {
							if sq == configSQ {
								m.selectedQualities[i] = struct{}{}
								break
							}
						}
					}
					m.state = stateQualities
				}
				return m, nil
			}

		case stateQualities:
			switch msg.String() {
			case "up", "k":
				if m.qualitiesCursor > 0 {
					m.qualitiesCursor--
				}
			case "down", "j":
				if m.qualitiesCursor < len(availableSoftwareQualities)-1 {
					m.qualitiesCursor++
				}
			case " ":
				_, ok := m.selectedQualities[m.qualitiesCursor]
				if ok {
					delete(m.selectedQualities, m.qualitiesCursor)
				} else {
					m.selectedQualities[m.qualitiesCursor] = struct{}{}
				}
			case "enter":
				if len(m.selectedQualities) > 0 {
					for k := range m.selectedQualities {
						m.softwareQualities = append(m.softwareQualities, availableSoftwareQualities[k])
					}
					m.state = stateCodePeriod
					return m, nil
				}
			}

		case stateCodePeriod:
			switch msg.String() {
			case "up", "k":
				if m.codePeriodCursor > 0 {
					m.codePeriodCursor--
				}
			case "down", "j":
				if m.codePeriodCursor < 1 {
					m.codePeriodCursor++
				}
			case "enter":
				m.isNewCodePeriod = m.codePeriodCursor == 0
				m.state = stateFetching
				return m, tea.Batch(m.spinner.Tick, fetchIssuesCmd(m.projectKey, m.token, m.softwareQualities, m.isNewCodePeriod))
			}

		case stateRemoveProject:
			switch msg.String() {
			case "up", "k":
				if m.removeProjectCursor > 0 {
					m.removeProjectCursor--
				}
			case "down", "j":
				if m.removeProjectCursor < len(m.config.Projects)-1 {
					m.removeProjectCursor++
				}
			case "enter":
				if len(m.config.Projects) > 0 {
					m.config.Projects = append(m.config.Projects[:m.removeProjectCursor], m.config.Projects[m.removeProjectCursor+1:]...)
					if err := saveConfig(m.config); err != nil {
					m.fetchErr = fmt.Errorf("failed to save config: %w", err)
					m.state = stateError
					return m, nil
				}

					availableProjects = []list.Item{}
					for _, p := range m.config.Projects {
						availableProjects = append(availableProjects, item(p))
					}
					if len(m.config.Projects) > 0 {
						availableProjects = append(availableProjects, item("- Remove Project"), item("+ Add New Project"))
					} else {
						availableProjects = append(availableProjects, item("+ Add New Project"))
					}
					m.projectList.SetItems(availableProjects)

					m.removeProjectCursor = 0
					m.state = stateProject
				}
				return m, nil
			case "esc":
				m.removeProjectCursor = 0
				m.state = stateProject
				return m, nil
			}

		case stateDone, stateError:
			if msg.String() == "q" || msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
				return m, tea.Quit
			}
		}
	}

	switch m.state {
	case stateURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)

	case stateNewProject:
		m.newProjectInput, cmd = m.newProjectInput.Update(msg)
		cmds = append(cmds, cmd)

	case stateToken:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
		cmds = append(cmds, cmd)

	case stateProject:
		m.projectList, cmd = m.projectList.Update(msg)
		cmds = append(cmds, cmd)

	case stateFetching:
		switch msg := msg.(type) {
		case issuesFetchedMsg:
			if msg.err != nil {
				m.fetchErr = msg.err
				m.state = stateError
				return m, nil
			}
			m.issues = msg.issues
			m.buildSummaryTable()
			var err error
			m.savedFile, err = exportToCSV(m.issues, m.projectKey)
			if err != nil {
				m.fetchErr = fmt.Errorf("failed to save CSV: %w", err)
				m.state = stateError
				return m, nil
			}
			m.state = statePrompt
			return m, nil
		case spinner.TickMsg:
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case stateDone:
		m.summaryTable, cmd = m.summaryTable.Update(msg)
		cmds = append(cmds, cmd)

	case statePrompt:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.promptCursor > 0 {
					m.promptCursor--
				}
			case "down", "j":
				if m.promptCursor < 1 {
					m.promptCursor++
				}
			case "enter":
				if m.promptCursor == 0 {
					m.projectKey = ""
					m.softwareQualities = []string{}
					m.selectedQualities = make(map[int]struct{})
					m.issues = nil
					m.qualitiesCursor = 0
					m.codePeriodCursor = 0
					m.isNewCodePeriod = true
					m.savedFile = ""
					m.promptCursor = 0
					m.state = stateProject
					return m, nil
				}
				if m.promptCursor == 1 {
					return m, tea.Quit
				}
			case "q":
				return m, tea.Quit
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) buildSummaryTable() {
	severityCounts := map[string]int{
		"HIGH":   0,
		"MEDIUM": 0,
		"LOW":    0,
	}

	for _, issue := range m.issues {
		severityCounts[issue.Severity]++
	}

	columns := []table.Column{
		{Title: "Impact Severity", Width: 20},
		{Title: "Count", Width: 10},
	}

	rows := []table.Row{
		{"HIGH", strconv.Itoa(severityCounts["HIGH"])},
		{"MEDIUM", strconv.Itoa(severityCounts["MEDIUM"])},
		{"LOW", strconv.Itoa(severityCounts["LOW"])},
		{"Total", strconv.Itoa(len(m.issues))},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(5),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63")).
		Bold(false)

	t.SetStyles(s)
	m.summaryTable = t
}

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

const asciiArt = `
 ▗▄▄▖ ▗▄▖ ▗▖  ▗▖ ▗▄▖ ▗▄▄▖  ▗▄▄▖▗▖ ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖ 
▐▌   ▐▌ ▐▌▐▛▚▖▐▌▐▌ ▐▌▐▌ ▐▌▐▌   ▐▌ ▐▌▐▌   ▐▌   ▐▌ ▐▌
 ▝▀▚▖▐▌ ▐▌▐▌ ▝▜▌▐▛▀▜▌▐▛▀▚▖ ▝▀▚▖▐▌ ▐▌▐▛▀▀▘▐▛▀▀▘▐▛▀▘ 
▗▄▄▞▘▝▚▄▞▘▐▌  ▐▌▐▌ ▐▌▐▌ ▐▌▗▄▄▞▘▐▙█▟▌▐▙▄▄▖▐▙▄▄▖▐▌   
                                                   
                                                   
                                                   `

func (m model) View() string {
	var b strings.Builder

	// Global Header
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(asciiArt))
	b.WriteString("\n\n")

	switch m.state {
	case stateURL:
		b.WriteString(subtitleStyle.Render("SonarQube Configuration"))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render("Enter your SonarQube URL (e.g., http://localhost:9000):"))
		b.WriteString("\n\n")
		b.WriteString(m.urlInput.View())
		if m.urlError != "" {
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(m.urlError))
		}
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("(Press Enter to continue, Esc to quit)"))

	case stateNewProject:
		b.WriteString(subtitleStyle.Render("SonarQube Configuration"))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render("Enter the SonarQube Project Key:"))
		b.WriteString("\n\n")
		b.WriteString(m.newProjectInput.View())
		if m.newProjectError != "" {
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(m.newProjectError))
		}
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("(Press Enter to continue, Esc to quit)"))

	case stateToken:
		b.WriteString(subtitleStyle.Render("Authentication Required"))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render("Token not found in .env or sonarsweep.json."))
		b.WriteString("\n\n")
		b.WriteString(m.tokenInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("(Press Enter to continue, Esc to quit)"))

	case stateProject:
		b.WriteString(subtitleStyle.Render("Select a Project"))
		b.WriteString("\n")
		b.WriteString(m.projectList.View())

	case stateQualities:
		b.WriteString(subtitleStyle.Render("Select Software Qualities"))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render("Space to toggle, Enter to confirm"))
		b.WriteString("\n\n")

		for i, choice := range availableSoftwareQualities {
			cursor := "  "
			if m.qualitiesCursor == i {
				cursor = cursorStyle.Render("> ")
			}

			checked := " "
			if _, ok := m.selectedQualities[i]; ok {
				checked = "•"
			}

			// Render row components separately
			box := ""
			if _, ok := m.selectedQualities[i]; ok {
				box = checkboxSelectedStyle.Render(fmt.Sprintf("[%s]", checked))
			} else {
				box = checkboxUnselectedStyle.Render(fmt.Sprintf("[%s]", checked))
			}

			text := choice
			if m.qualitiesCursor == i {
				text = highlightStyle.Render(text)
			}

			// Join them exactly without nesting ANSI
			line := fmt.Sprintf("%s%s %s\n", cursor, box, text)
			b.WriteString(line)
		}

		if len(m.selectedQualities) == 0 {
			b.WriteString(errorStyle.Render("\n(You must select at least one quality)"))
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: Navigate • Space: Toggle • Enter: Confirm • Esc: Quit"))

	case stateCodePeriod:
		b.WriteString(subtitleStyle.Render("Select Code Period"))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render("Up/Down to navigate, Enter to confirm"))
		b.WriteString("\n\n")

		choices := []string{"New Code (Default)", "Overall Code"}
		for i, choice := range choices {
			cursor := "  "
			if m.codePeriodCursor == i {
				cursor = cursorStyle.Render("> ")
			}

			text := choice
			if m.codePeriodCursor == i {
				text = highlightStyle.Render(text)
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, text))
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Confirm • Esc: Quit"))

	case stateRemoveProject:
		b.WriteString(dangerActionStyle.Render("Remove Project"))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render("Select a project to remove and press Enter"))
		b.WriteString("\n\n")

		for i, proj := range m.config.Projects {
			cursor := "  "
			if m.removeProjectCursor == i {
				cursor = cursorStyle.Render("> ")
			}

			text := proj
			if m.removeProjectCursor == i {
				text = dangerActionStyle.Render(proj)
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, text))
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Remove • Esc: Cancel"))

	case stateFetching:
		b.WriteString(subtitleStyle.Render("Fetching Issues..."))
		b.WriteString("\n\n")

		spinView := fmt.Sprintf("%s Fetching issues for %s...", m.spinner.View(), highlightStyle.Render(m.projectKey))
		b.WriteString(spinView)

	case stateError:
		b.WriteString(errorStyle.Render("Error occurred"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%v", m.fetchErr))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press q to quit."))

	case stateDone:
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Summary for %s", m.projectKey)))
		b.WriteString("\n")
		b.WriteString(subtextStyle.Render(fmt.Sprintf("Qualities: %s", strings.Join(m.softwareQualities, ", "))))
		b.WriteString("\n\n")
		b.WriteString(m.summaryTable.View())
		b.WriteString("\n\n")
		if m.savedFile != "" {
			b.WriteString(successStyle.Render(fmt.Sprintf("Export complete! Data saved to: %s", m.savedFile)))
		} else {
			b.WriteString(errorStyle.Render("Failed to save the output file."))
		}

	case statePrompt:
		b.WriteString(subtitleStyle.Render("Export Successful!"))
		b.WriteString("\n")
		if m.savedFile != "" {
			b.WriteString(subtextStyle.Render(fmt.Sprintf("Data saved to: %s", m.savedFile)))
		} else {
			b.WriteString(errorStyle.Render("Failed to save the output file."))
		}
		b.WriteString("\n\n")
		b.WriteString(subtextStyle.Render("What would you like to do next?"))
		b.WriteString("\n\n")

		choices := []string{"Export another project", "Exit SonarSweep"}
		for i, choice := range choices {
			cursor := "  "
			if m.promptCursor == i {
				cursor = cursorStyle.Render("> ")
			}

			text := choice
			if m.promptCursor == i {
				text = highlightStyle.Render(text)
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, text))
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Confirm • q: Quit"))
	}
	if m.state == stateError {
		b.WriteString(errorStyle.Render("Error occurred"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%v", m.fetchErr))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press q to quit."))
	}

	return appStyle.Render(b.String())
}

func main() {
	var (
		help         bool
		version      bool
		reset        bool
		viewConfig   bool
		listProjects bool
		addProject   string
		exportPath   string
		dryRun       bool
		quiet        bool
	)

	flag.BoolVar(&help, "help", false, "Show this help message and exit")
	flag.BoolVar(&help, "h", false, "Show this help message and exit (shorthand)")
	flag.BoolVar(&version, "version", false, "Show version information and exit")
	flag.BoolVar(&version, "v", false, "Show version information and exit (shorthand)")
	flag.BoolVar(&reset, "reset", false, "Reset the saved URL and Projects from config")
	flag.BoolVar(&viewConfig, "view-config", false, "Print current configuration and exit")
	flag.BoolVar(&listProjects, "list-projects", false, "List all saved projects and exit")
	flag.StringVar(&addProject, "add-project", "", "Add a project to the configuration")
	flag.StringVar(&exportPath, "export", "", "Override the CSV export path")
	flag.BoolVar(&dryRun, "dry-run", false, "Fetch issues but skip CSV export")
	flag.BoolVar(&quiet, "quiet", false, "Run in headless mode (no TUI)")
	flag.BoolVar(&quiet, "q", false, "Run in headless mode (no TUI) (shorthand)")
	flag.StringVar(&configPath, "config", "", "Use a different configuration file")
	flag.StringVar(&configPath, "c", "", "Use a different configuration file (shorthand)")

	flag.Usage = printHelp
	flag.Parse()

	if help {
		printHelp()
		os.Exit(0)
	}

	if version {
		fmt.Printf("SonarSweep v%s\n", VERSION)
		os.Exit(0)
	}

	if reset {
		cfg := loadConfig()
		cfg.SonarURL = ""
		cfg.Projects = []string{}
		if err := saveConfig(cfg); err != nil {
			fmt.Printf("Failed to reset config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration reset successfully.")
		os.Exit(0)
	}

	if viewConfig {
		cfg := loadConfig()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		os.Exit(0)
	}

	if listProjects {
		cfg := loadConfig()
		if len(cfg.Projects) == 0 {
			fmt.Println("No projects saved. Run the TUI to add projects.")
		} else {
			for _, p := range cfg.Projects {
				fmt.Println(p)
			}
		}
		os.Exit(0)
	}

	isHeadless := quiet || dryRun || exportPath != ""

	if addProject != "" {
		cfg := loadConfig()
		exists := false
		for _, p := range cfg.Projects {
			if p == addProject {
				exists = true
				break
			}
		}
		if !exists {
			cfg.Projects = append(cfg.Projects, addProject)
			if err := saveConfig(cfg); err != nil {
				if !quiet {
					fmt.Printf("Failed to save config: %v\n", err)
				}
				os.Exit(1)
			}
			if !quiet && !isHeadless {
				fmt.Printf("Project '%s' added successfully.\n", addProject)
			}
		} else if !quiet && !isHeadless {
			fmt.Printf("Project '%s' already exists.\n", addProject)
		}

		if !isHeadless {
			os.Exit(0)
		}
	}

	if isHeadless {
		godotenv.Load()
		cfg := loadConfig()

		if cfg.SonarURL == "" {
			if !quiet {
				fmt.Fprintln(os.Stderr, "Error: SonarQube URL is not configured. Run the TUI first to set it.")
			}
			os.Exit(1)
		}
		SONAR_URL = cfg.SonarURL

		token := os.Getenv("USER_TOKEN")
		if token == "" {
			token = cfg.Token
		}
		if token == "" {
			if !quiet {
				fmt.Fprintln(os.Stderr, "Error: Authentication token is missing. Set USER_TOKEN env var or run the TUI to save it.")
			}
			os.Exit(1)
		}

		projectKey := addProject
		if projectKey == "" {
			if len(cfg.Projects) == 0 {
				if !quiet {
					fmt.Fprintln(os.Stderr, "Error: No projects configured. Use --add-project <key> to specify one.")
				}
				os.Exit(1)
			}
			projectKey = cfg.Projects[0]
		}

		if exportPath != "" {
			cliExportPath = exportPath
		}

		if !quiet {
			fmt.Printf("Fetching issues for project: %s\n", projectKey)
		}

		issues, err := fetchIssues(projectKey, token, cfg.SoftwareQualities, true)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
			}
			os.Exit(1)
		}

		if dryRun {
			if !quiet {
				severityCounts := map[string]int{"HIGH": 0, "MEDIUM": 0, "LOW": 0}
				for _, issue := range issues {
					severityCounts[issue.Severity]++
				}
				fmt.Println("\nDry Run Summary:")
				fmt.Printf("Total Issues: %d\n", len(issues))
				fmt.Printf("HIGH:   %d\n", severityCounts["HIGH"])
				fmt.Printf("MEDIUM: %d\n", severityCounts["MEDIUM"])
				fmt.Printf("LOW:    %d\n", severityCounts["LOW"])
			}
			os.Exit(0)
		}

		savedFile, err := exportToCSV(issues, projectKey)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Error exporting CSV: %v\n", err)
			}
			os.Exit(1)
		}

		if !quiet {
			fmt.Printf("Export complete! Data saved to: %s\n", savedFile)
		}
		os.Exit(0)
	}

	// No flags, proceed with TUI
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
