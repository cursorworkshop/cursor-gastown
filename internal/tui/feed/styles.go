// Package feed provides a TUI for the Gas Town activity feed.
package feed

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/steveyegge/gastown/internal/constants"
)

// Color palette
var (
	colorPrimary   = lipgloss.Color("12")  // Blue
	colorSuccess   = lipgloss.Color("10")  // Green
	colorWarning   = lipgloss.Color("11")  // Yellow
	colorError     = lipgloss.Color("9")   // Red
	colorDim       = lipgloss.Color("8")   // Gray
	colorHighlight = lipgloss.Color("14")  // Cyan
	colorAccent    = lipgloss.Color("13")  // Magenta
)

// Styles for the feed TUI
var (
	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	FilterStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Agent tree styles
	TreePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Padding(0, 1)

	RigStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	RoleStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	AgentNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	AgentActiveStyle = lipgloss.NewStyle().
				Foreground(colorSuccess)

	AgentIdleStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Event stream styles
	StreamPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDim).
				Padding(0, 1)

	TimestampStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	EventCreateStyle = lipgloss.NewStyle().
				Foreground(colorSuccess)

	EventUpdateStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	EventCompleteStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	EventFailStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	EventDeleteStyle = lipgloss.NewStyle().
				Foreground(colorWarning)

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(colorDim).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Focus indicator
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Role icons - uses centralized icons from constants package
	RoleIcons = map[string]string{
		constants.RoleMayor:    constants.IconMayor,
		constants.RoleWitness:  constants.IconWitness,
		constants.RoleRefinery: constants.IconRefinery,
		constants.RoleCrew:     constants.IconCrew,
		constants.RolePolecat:  constants.IconPolecat,
		constants.RoleDeacon:   constants.IconDeacon,
	}

	// MQ event styles
	EventMergeStartedStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	EventMergedStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	EventMergeFailedStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	EventMergeSkippedStyle = lipgloss.NewStyle().
				Foreground(colorWarning)

	// Event symbols
	EventSymbols = map[string]string{
		"create":   "+",
		"update":   ">",
		"complete": "*",
		"fail":     "x",
		"delete":   "-",
		"pin":      "^",
		// Witness patrol events
		"patrol_started":  constants.IconWitness,
		"patrol_complete": "*",
		"polecat_checked": ".",
		"polecat_nudged":  "!",
		"escalation_sent": "^",
		// Merge events
		"merge_started": "#",
		"merged":        "*",
		"merge_failed":  "x",
		"merge_skipped": "-",
		// General gt events
		"sling":   ">",
		"hook":    "[hook]",
		"unhook":  "[unhook]",
		"handoff": "[handoff]",
		"done":    "*",
		"mail":    "@",
		"spawn":   "[spawn]",
		"kill":    "[kill]",
		"nudge":   "!",
		"boot":    "[boot]",
		"halt":    "[halt]",
	}
)
