package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ibfavas/netpulse/internal/config"
)

var (
	CyberBlack  = lipgloss.Color("#0F1015")
	NeonPurple  = lipgloss.Color("#FF007F")
	CyberLime   = lipgloss.Color("#00FF66")
	NeonAmber   = lipgloss.Color("#FF9F00")
	GlitchRed   = lipgloss.Color("#FF003C")
	MatrixGray  = lipgloss.Color("#565E75")
	SubduedCyan = lipgloss.Color("#4EA8DE")
)

func getBorderColor(focused bool, hasLoss bool, highLatency bool) lipgloss.Color {
	if focused {
		return NeonPurple
	}
	if hasLoss {
		return GlitchRed
	}
	if highLatency {
		return NeonAmber
	}
	return MatrixGray
}

func GetBoxStyle(focused, hasLoss, highLatency bool, width, height int) lipgloss.Style {
	s := lipgloss.NewStyle().
		BorderForeground(getBorderColor(focused, hasLoss, highLatency)).
		Width(width).
		Height(height)

	if focused {
		s = s.BorderStyle(lipgloss.DoubleBorder())
	} else {
		s = s.BorderStyle(lipgloss.RoundedBorder())
	}
	return s
}

func RenderBadge(text string, isGood bool) string {
	bg := CyberLime
	fg := CyberBlack
	if !isGood {
		bg = GlitchRed
		fg = lipgloss.Color("#FFFFFF")
	}
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

func formatVal(val float64, cfg *config.Config) string {
	if val < 0 {
		return lipgloss.NewStyle().Foreground(GlitchRed).Bold(true).Render("TIMEOUT")
	}
	s := fmt.Sprintf("%7.2fms", val)
	if val < 20 {
		return lipgloss.NewStyle().Foreground(CyberLime).Render(s)
	} else if val <= 80 {
		return lipgloss.NewStyle().Foreground(NeonAmber).Render(s)
	}
	return lipgloss.NewStyle().Foreground(GlitchRed).Render(s)
}

func RenderTitle(title string) string {
	return lipgloss.NewStyle().
		Foreground(SubduedCyan).
		Bold(true).
		Render(fmt.Sprintf("▰▰ [ %s ] ▰▰", title))
}
