package main

import "github.com/charmbracelet/lipgloss"

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

const asciiArt = `
 ▗▄▄▖ ▗▄▖ ▗▖  ▗▖ ▗▄▖ ▗▄▄▖  ▗▄▄▖▗▖ ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖ 
▐▌   ▐▌ ▐▌▐▛▚▖▐▌▐▌ ▐▌▐▌ ▐▌▐▌   ▐▌ ▐▌▐▌   ▐▌   ▐▌ ▐▌
 ▝▀▚▖▐▌ ▐▌▐▌ ▝▜▌▐▛▀▜▌▐▛▀▚▖ ▝▀▚▖▐▌ ▐▌▐▛▀▀▘▐▛▀▀▘▐▛▀▘ 
▗▄▄▞▘▝▚▄▞▘▐▌  ▐▌▐▌ ▐▌▐▌ ▐▌▗▄▄▞▘▐▙█▟▌▐▙▄▄▖▐▙▄▄▖▐▌   
                                                   
                                                   
                                                   `
