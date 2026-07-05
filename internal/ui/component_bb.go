package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func computeMatrix(history []float64) (min, max, avg, jitter float64) {
	if len(history) == 0 {
		return 0, 0, 0, 0
	}
	min = -1.0
	max = -1.0
	sum := 0.0
	jitterSum := 0.0
	jCount := 0.0
	validCount := 0.0
	for i, v := range history {
		if v < 0 {
			continue // skip timeouts
		}
		if min < 0 || v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
		validCount++
		if i > 0 && history[i-1] >= 0 {
			diff := v - history[i-1]
			if diff < 0 {
				diff = -diff
			}
			jitterSum += diff
			jCount++
		}
	}
	if validCount > 0 {
		avg = sum / validCount
	}
	if jCount > 0 {
		jitter = jitterSum / jCount
	}
	if min < 0 {
		min = 0
	}
	if max < 0 {
		max = 0
	}
	return
}

func RenderBackbones(m Model, height int) string {
	boxWidth := m.width - 2
	if boxWidth < 20 {
		boxWidth = 20
	}

	var lines []string
	hasLoss := false
	highLatency := false

	maxItems := height - 2
	if maxItems < 1 {
		maxItems = 1
	}

	for i, b := range m.backbones {
		if i >= maxItems {
			break
		}
		target := b.Target
		if len(target) > 25 {
			target = target[:25]
		}

		var icon string = "◈"
		if b.Error != nil || b.Loss > 0 {
			icon = "▲"
		}

		var targetStr string
		if m.focus == FocusBackbones && i == m.bbCursor {
			targetStr = lipgloss.NewStyle().Foreground(CyberBlack).Background(NeonPurple).Bold(true).Render(fmt.Sprintf("[ %s %-23s ► ]", icon, target))
		} else {
			targetStr = fmt.Sprintf("  %s %-23s   ", icon, target)
		}

		val := -1.0
		if b.Error == nil {
			val = float64(b.Latency.Microseconds()) / 1000.0
			if val > float64(m.cfg.Daemon.AlertThreshold) {
				highLatency = true
			}
		} else {
			hasLoss = true
		}
		if b.Loss > 0 {
			hasLoss = true
		}

		min, max, avg, jitter := computeMatrix(m.bbHistory[b.Target])

		matrix := lipgloss.NewStyle().Foreground(MatrixGray).Render("│ Cur: ") +
			formatVal(val, m.cfg) +
			lipgloss.NewStyle().Foreground(MatrixGray).Render(fmt.Sprintf(" │ Min: %6.2fms │ Max: %6.2fms │ Avg: %6.2fms │ Loss: %3.0f%% │ Jitter: %6.2fms │", min, max, avg, b.Loss, jitter))

		sparkWidth := boxWidth - lipgloss.Width(targetStr) - lipgloss.Width(matrix) - 4
		if sparkWidth < 5 {
			sparkWidth = 5
		}
		spark := GenerateSparkline(m.bbHistory[b.Target], sparkWidth)
		lines = append(lines, fmt.Sprintf("%s %s %s", targetStr, matrix, spark))
	}

	if len(m.backbones) == 0 {
		lines = append(lines, "No backbone targets configured.")
	}

	badge := RenderBadge("ONLINE", true)
	if hasLoss {
		badge = RenderBadge("FAULT", false)
	} else if highLatency {
		badge = RenderBadge("WARN", false)
	}

	titleText := RenderTitle("SYSTEM: EXTERNAL_TRANSIT")
	title := lipgloss.JoinHorizontal(lipgloss.Center, titleText, " ", badge)

	minHeight := len(lines) + 3
	if height < minHeight {
		height = minHeight
	}

	return GetBoxStyle(m.focus == FocusBackbones, hasLoss, highLatency, boxWidth, height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(lines, "\n")))
}
