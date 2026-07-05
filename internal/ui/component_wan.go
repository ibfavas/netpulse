package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func RenderWAN(m Model, width int, height int) string {
	boxWidth := width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	titleText := RenderTitle("NODE: WAN_TELEMETRY")
	badge := RenderBadge("SYNCED", true)
	if m.wanMeta != nil {
		badge = RenderBadge("RESOLVED", true)
	}
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText, " ", badge)

	content := "  Awaiting telemetry data..."
	if m.wanMeta != nil {
		org := m.wanMeta.Org
		ip := m.wanMeta.IP
		city := m.wanMeta.City
		region := m.wanMeta.Region

		if m.isDemo {
			org = "*****************"
			ip = "***.***.***.***"
			city = "********"
			region = "********"
		}

		if len(org) > 25 {
			org = org[:25]
		}
		content = fmt.Sprintf("  ISP:  %s\n  IP:   %-15s\n  Loc:  %s, %s\n  Ctr:  %s",
			org, ip, city, region, m.wanMeta.Country)
	}

	return GetBoxStyle(m.focus == FocusWAN, false, false, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}
