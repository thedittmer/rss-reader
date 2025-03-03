package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Core color palette inspired by pnpm with additional vibrant colors
	primaryColor   = lipgloss.Color("#0969DA") // GitHub blue
	secondaryColor = lipgloss.Color("#8250DF") // Purple
	accentColor    = lipgloss.Color("#2DA44E") // Green
	warningColor   = lipgloss.Color("#D29922") // Orange
	errorColor     = lipgloss.Color("#CF222E") // Red
	textColor      = lipgloss.Color("#FFFFFF") // White
	dimColor       = lipgloss.Color("#6E7681") // Gray
	linkColor      = lipgloss.Color("#58A6FF") // Light blue
	scoreColor     = lipgloss.Color("#F778BA") // Pink
	titleColor     = lipgloss.Color("#39D353") // Bright green
	dateColor      = lipgloss.Color("#A371F7") // Light purple
	sourceColor    = lipgloss.Color("#FFA657") // Light orange
	bgColor        = lipgloss.Color("#1F2328") // Dark background
	selectedBg     = lipgloss.Color("#2D333B") // Selected item background

	// Enhanced styles with consistent color usage
	HeaderStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(1, 0).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accentColor)

	CommandStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	ArrowStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			SetString("│ ")

	SuccessStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	TextStyle = lipgloss.NewStyle().
			Foreground(textColor)

	DimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	LinkStyle = lipgloss.NewStyle().
			Foreground(linkColor).
			Underline(true)

	ScoreStyle = lipgloss.NewStyle().
			Foreground(scoreColor).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(titleColor).
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1)

	DateStyle = lipgloss.NewStyle().
			Foreground(dateColor).
			Italic(true)

	SourceStyle = lipgloss.NewStyle().
			Foreground(sourceColor).
			Bold(true)

	SectionStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Padding(0, 0, 1, 0).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accentColor)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Background(selectedBg).
			Bold(true).
			SetString("▶")

	UnselectedStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			SetString(" ")

	KeyStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accentColor)

	// New styles for enhanced UI elements
	MenuItemStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(bgColor).
			Padding(0, 1).
			MarginLeft(2)

	StatusStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Background(selectedBg).
			Padding(0, 1).
			Bold(true)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(scoreColor).
			Bold(true).
			Underline(true)

	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1)
)
