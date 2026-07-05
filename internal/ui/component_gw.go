package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func RenderGateway(m Model, width int, height int) string {
	boxWidth := width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	val := float64(m.gwPing.Latency.Microseconds()) / 1000.0
	hasLoss := m.gwPing.Loss > 0 || m.gwPing.Error != nil
	highLatency := val > 80 && !hasLoss

	if hasLoss {
		val = -1
	}

	badge := RenderBadge("ONLINE", true)
	if hasLoss {
		badge = RenderBadge("LOSS DETECTED", false)
	} else if highLatency {
		badge = RenderBadge("HIGH LATENCY", false)
	}

	titleText := RenderTitle("MODULE: GATEWAY_CORE")
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText, " ", badge)

	target := m.gwPing.Target
	if target == "" {
		target = "Pending..."
	}
	if len(target) > 15 {
		target = target[:15]
	}

	sparkWidth := boxWidth - 30
	if sparkWidth < 5 {
		sparkWidth = 5
	}
	spark := GenerateSparkline(m.gwHistory, sparkWidth)

	m.prog.Width = 15
	progBar := m.prog.ViewAs(m.gwPing.Loss / 100.0)

	var lossStr string
	if hasLoss {
		progBar = lipgloss.NewStyle().Foreground(GlitchRed).Render(progBar)
		lossStr = lipgloss.NewStyle().Foreground(GlitchRed).Bold(true).Render(fmt.Sprintf("[ %s ] %.0f%% DETECTED", progBar, m.gwPing.Loss))
	} else {
		progBar = lipgloss.NewStyle().Foreground(MatrixGray).Render(progBar)
		lossStr = lipgloss.NewStyle().Foreground(SubduedCyan).Render(fmt.Sprintf("[ %s ] 0%% SYSTEM NORMAL", progBar))
	}

	content := fmt.Sprintf("  Target:  %-15s\n  Latency: %s %s\n  Link:    ⚡ SECURE\n  Loss:    %s",
		target,
		formatVal(val, m.cfg), spark,
		lossStr)

	return GetBoxStyle(m.focus == FocusGateway, hasLoss, highLatency, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}
