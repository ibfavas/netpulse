package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderDNS(m Model, width int, height int) string {
	boxWidth := width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	var lines []string
	hasLoss := false
	highLatency := false

	maxItems := height - 4
	if maxItems < 1 {
		maxItems = 1
	}

	for i, d := range m.dns {
		if i >= maxItems {
			break
		}
		provider := d.Provider
		if len(provider) > 25 {
			provider = provider[:25]
		}

		var icon string = "◈"
		if d.Error != nil {
			icon = "▲"
		}

		var providerStr string
		if m.focus == FocusDNS && i == m.dnsCursor {
			providerStr = lipgloss.NewStyle().Foreground(CyberBlack).Background(NeonPurple).Bold(true).Render(fmt.Sprintf("[ %s %-23s ► ]", icon, provider))
		} else {
			providerStr = fmt.Sprintf("  %s %-23s   ", icon, provider)
		}

		if d.Error != nil {
			errStr := lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.Bad)).Render("ERROR")
			lines = append(lines, fmt.Sprintf("%s %s", providerStr, errStr))
			hasLoss = true
			continue
		}

		val := float64(d.Latency.Microseconds()) / 1000.0
		if val > float64(m.cfg.Daemon.AlertThreshold) {
			highLatency = true
		}

		sparkWidth := boxWidth - 45
		if sparkWidth < 5 {
			sparkWidth = 5
		}
		spark := GenerateSparkline(m.dnsHistory[d.Provider], sparkWidth)
		lines = append(lines, fmt.Sprintf("%s %s %s", providerStr, formatVal(val, m.cfg), spark))
	}

	if len(m.dns) == 0 {
		lines = append(lines, "No DNS targets configured.")
	}

	titleText := RenderTitle("SUBSYSTEM: DNS_MATRIX")
	badge := RenderBadge("OPERATIONAL", true)
	if hasLoss {
		badge = RenderBadge("FAULT", false)
	} else if highLatency {
		badge = RenderBadge("WARN", false)
	}
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText, " ", badge)

	return GetBoxStyle(m.focus == FocusDNS, hasLoss, highLatency, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(lines, "\n")))
}
