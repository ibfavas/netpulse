package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/ibfavas/netpulse/internal/diagnostics"
)

func formatSpeed(bps float64) string {
	if bps < 1024 {
		return fmt.Sprintf("%6.1f B/s", bps)
	} else if bps < 1024*1024 {
		return fmt.Sprintf("%6.1f KB/s", bps/1024)
	}
	return fmt.Sprintf("%6.1f MB/s", bps/1024/1024)
}

func RenderIface(m Model, width int, height int) string {
	boxWidth := width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	titleText := RenderTitle("SENSOR: LOCAL_HARDWARE")
	badge := RenderBadge("ACTIVE", true)
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText, " ", badge)

	content := "No interfaces detected."
	if len(m.ifaces) > 0 {
		var active *diagnostics.IfaceStats
		for _, iface := range m.ifaces {
			if iface.IP != "" {
				copyIface := iface
				active = &copyIface
				break
			}
		}
		if active == nil {
			active = &m.ifaces[0]
		}

		mac := active.MAC
		if mac == "" {
			mac = "Unknown"
		}

		displayIP := active.IP
		if m.isDemo {
			mac = "xx:xx:xx:xx:xx:xx"
			displayIP = "xxx.xxx.xxx.xxx"
		}

		rxSparkWidth := boxWidth - 30
		if rxSparkWidth < 5 {
			rxSparkWidth = 5
		}
		rxSpark := lipgloss.NewStyle().Foreground(CyberLime).Render(GenerateSparkline(m.rxHistory, rxSparkWidth))
		txSpark := lipgloss.NewStyle().Foreground(NeonPurple).Render(GenerateSparkline(m.txHistory, rxSparkWidth))

		content = fmt.Sprintf("Iface:  %-15s\nIP:     %-15s\nMAC:    %-15s\nRX: %-10s %s\nTX: %-10s %s",
			active.Name,
			displayIP,
			mac,
			formatSpeed(active.RXSpeed), rxSpark,
			formatSpeed(active.TXSpeed), txSpark)
	}

	return GetBoxStyle(m.focus == FocusIface, false, false, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}
