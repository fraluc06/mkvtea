package ui

import "github.com/charmbracelet/lipgloss"

// Color Palette - Catppuccin Mocha
// https://github.com/catppuccin/catppuccin
const (
	colorBg            = "#1e1e2e" // Mocha base
	colorBgLight       = "#313244" // Mocha surface0
	colorBgLighter     = "#45475a" // Mocha surface1
	colorText          = "#cdd6f4" // Mocha text
	colorTextDim       = "#a6adc8" // Mocha subtext1
	colorPrimaryPurple = "#b892f0" // Mocha mauve
	colorAccentCyan    = "#89dceb" // Mocha sky
	colorAccentBlue    = "#89b4fa" // Mocha blue
	colorSuccessGreen  = "#a6e3a1" // Mocha green
	colorWarningOrange = "#fab387" // Mocha peach
	colorErrorRed      = "#f38ba8" // Mocha red/flamingo
	colorAccentPink    = "#f5c2e7" // Mocha pink
)

var (
	// Base styles
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorTextDim))
	docStyle    = lipgloss.NewStyle().Margin(0, 0).Foreground(lipgloss.Color(colorText))

	// Main header - bold and luminous
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Background(lipgloss.Color(colorPrimaryPurple)).
			Padding(0, 2).
			Bold(true).
			Align(lipgloss.Center)

	// Subtitle with accent color
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccentCyan)).
			Bold(true)

	// Box for sections
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPrimaryPurple)).
			Padding(0, 1).
			MarginBottom(0)

	// Stats box with accent color
	statsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorAccentCyan)).
			Padding(0, 1).
			MarginBottom(0).
			Background(lipgloss.Color(colorBgLight))

	// Success - neon green
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccessGreen)).
			Bold(true)

	// Error - magenta red
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorErrorRed)).
			Bold(true)

	// Warning - Orange
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWarningOrange))

	// Progress bar color
	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccentBlue)).
			Bold(true)

	// Selected style
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBg)).
			Background(lipgloss.Color(colorSuccessGreen)).
			Bold(true).
			Padding(0, 1)

	// Focused style
	focusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccentCyan)).
			Background(lipgloss.Color(colorBgLight)).
			Padding(0, 1)

	// Focused + Selected
	focusedSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBg)).
				Background(lipgloss.Color(colorAccentCyan)).
				Bold(true).
				Padding(0, 1)
)
