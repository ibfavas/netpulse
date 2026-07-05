package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func RenderMTR(m Model) string {
	cfg := m.cfg
	boxWidth := m.width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	headerText := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("236")).
		Padding(0, 2).
		MarginBottom(1).
		Width(m.width).
		Render("NetPulse: MTR Mode")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cfg.Theme.Title)).Padding(0, 1)

	content := ""
	if m.mtrLoading {
		content = "Running full traceroute grid... please wait."
	} else if m.mtrErr != nil {
		content = fmt.Sprintf("MTR Error: %v", m.mtrErr)
	} else {
		var lines []string
		lines = append(lines, fmt.Sprintf("%-5s %-20s %s", "HOP", "IP ADDRESS", "LATENCY"))

		ruleWidth := boxWidth - 5
		if ruleWidth < 0 {
			ruleWidth = 10
		}
		lines = append(lines, strings.Repeat("-", ruleWidth))

		for _, hop := range m.mtrHops {
			if hop.Lost {
				lines = append(lines, fmt.Sprintf("%-5d %-20s %s", hop.TTL, "* * *", lipgloss.NewStyle().Foreground(GlitchRed).Render("TIMEOUT")))
			} else {
				lines = append(lines, fmt.Sprintf("%-5d %-20s %s", hop.TTL, hop.IP, formatVal(float64(hop.Latency.Microseconds())/1000.0, cfg)))
			}
		}
		content = strings.Join(lines, "\n")
	}

	boxHeight := m.height - 4
	if boxHeight < 20 {
		boxHeight = 20
	}

	mtrBox := GetBoxStyle(true, false, false, boxWidth, boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("MTR / Interactive Traceroute"), "", content))

	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1).
		Render("Esc: Back to Dashboard • Q/Ctrl+C: Quit")

	return lipgloss.JoinVertical(lipgloss.Left, headerText, mtrBox, footer)
}
