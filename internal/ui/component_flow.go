package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderFlow(m Model, height int) string {
	boxWidth := m.width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	titleText := RenderTitle("MODULE: PACKET_FLOW_OBSERVER")
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText)

	var lines []string
	if len(m.ifaces) > 0 {
		iface := m.ifaces[0]
		rxSpeed := float64(iface.RXSpeed)
		txSpeed := float64(iface.TXSpeed)

		rxSparkWidth := boxWidth - 55
		if rxSparkWidth < 5 {
			rxSparkWidth = 5
		}
		rxSpark := lipgloss.NewStyle().Foreground(CyberLime).Render(GenerateSparkline(m.rxHistory, rxSparkWidth))
		txSpark := lipgloss.NewStyle().Foreground(NeonPurple).Render(GenerateSparkline(m.txHistory, rxSparkWidth))

		lines = append(lines, fmt.Sprintf("  NETWORK INGRESS  [RX]: %s  %-10s [⚡ MAX LINK]", rxSpark, formatSpeed(rxSpeed)))
		lines = append(lines, fmt.Sprintf("  NETWORK EGRESS   [TX]: %s  %-10s [🟢 UNTHROTTLED]", txSpark, formatSpeed(txSpeed)))
	} else {
		lines = append(lines, "  No flow telemetry available.")
	}

	return GetBoxStyle(false, false, false, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(lines, "\n")))
}
