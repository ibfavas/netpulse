package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderLogs(m Model, height int) string {
	boxWidth := m.width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	titleText := RenderTitle("SUBSYSTEM: TELEMETRY_EVENT_LOG")
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText)

	contentHeight := height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	var lines []string
	startIdx := len(m.logs) - contentHeight
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(m.logs); i++ {
		line := m.logs[i]
		if strings.HasPrefix(line, "[") && len(line) > 10 {
			timestamp := lipgloss.NewStyle().Foreground(MatrixGray).Render(line[1:9] + " │")
			rest := line[11:]
			var level, msgStr string

			if strings.HasPrefix(rest, "ALERT") {
				level = lipgloss.NewStyle().Foreground(GlitchRed).Bold(true).Render("● ALERT ")
				msgStr = lipgloss.NewStyle().Foreground(GlitchRed).Render(rest[5:])
			} else if strings.HasPrefix(rest, "WARN ") {
				level = lipgloss.NewStyle().Foreground(NeonAmber).Bold(true).Render("▲ WARN  ")
				msgStr = lipgloss.NewStyle().Foreground(NeonAmber).Render(rest[5:])
			} else if strings.HasPrefix(rest, "USER ") {
				level = lipgloss.NewStyle().Foreground(CyberLime).Bold(true).Render("◆ USER  ")
				msgStr = lipgloss.NewStyle().Foreground(SubduedCyan).Render(rest[5:])
			} else if strings.HasPrefix(rest, "INFO ") {
				level = lipgloss.NewStyle().Foreground(SubduedCyan).Bold(true).Render("◈ INFO  ")
				msgStr = lipgloss.NewStyle().Foreground(SubduedCyan).Render(rest[5:])
			} else if strings.HasPrefix(rest, "BGP  ") {
				level = lipgloss.NewStyle().Foreground(NeonPurple).Bold(true).Render("◈ BGP   ")
				msgStr = lipgloss.NewStyle().Foreground(SubduedCyan).Render(rest[5:])
			} else {
				level = rest
			}

			line = fmt.Sprintf("  %s %s %s", timestamp, level, msgStr)
		} else {
			line = "  " + lipgloss.NewStyle().Foreground(MatrixGray).Render(line)
		}
		lines = append(lines, line)
	}

	return GetBoxStyle(false, false, false, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(lines, "\n")))
}
