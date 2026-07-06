package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ibfavas/netpulse/internal/config"
	"github.com/ibfavas/netpulse/internal/diagnostics"
)

type tickMsg time.Time

type metricsMsg struct {
	gwPing    diagnostics.PingResult
	backbones []diagnostics.PingResult
	dns       []diagnostics.DNSResult
	ifaces    []diagnostics.IfaceStats
}

type wanMsg struct {
	meta *diagnostics.WANMetadata
	err  error
}

type tracerouteMsg struct {
	hops []diagnostics.Hop
	err  error
}

type FocusPanel int

const (
	FocusGateway FocusPanel = iota
	FocusDNS
	FocusIface
	FocusWAN
	FocusBackbones
)

type Model struct {
	cfg       *config.Config
	isDemo    bool
	gwPing    diagnostics.PingResult
	backbones []diagnostics.PingResult
	dns       []diagnostics.DNSResult
	ifaces    []diagnostics.IfaceStats
	wanMeta   *diagnostics.WANMetadata

	gwHistory  []float64
	bbHistory  map[string][]float64
	dnsHistory map[string][]float64
	rxHistory  []float64
	txHistory  []float64

	logs      []string
	nodeState map[string]string

	paused      bool
	extendedDNS bool
	mtrMode     bool
	mtrHops     []diagnostics.Hop
	mtrErr      error
	mtrLoading  bool

	focus     FocusPanel
	dnsCursor int
	bbCursor  int

	tickRate time.Duration
	prog     progress.Model

	textInput    textinput.Model
	inputActive  bool
	renameActive bool
	footerError  string

	termWidth  int
	termHeight int
	width      int
	height     int
}

func InitialModel(cfg *config.Config, isDemo bool) Model {
	prog := progress.New(
		progress.WithSolidFill("#FF003C"),
		progress.WithoutPercentage(),
	)
	prog.Full = '█'
	prog.Empty = '░'
	prog.Width = 15

	ti := textinput.New()
	ti.Placeholder = "Enter IP to track (e.g. 1.1.1.1)"
	ti.CharLimit = 30
	ti.Width = 30

	return Model{
		cfg:        cfg,
		isDemo:     isDemo,
		backbones:  make([]diagnostics.PingResult, 0),
		dns:        make([]diagnostics.DNSResult, 0),
		bbHistory:  make(map[string][]float64),
		dnsHistory: make(map[string][]float64),
		rxHistory:  make([]float64, 0),
		txHistory:  make([]float64, 0),
		logs:       []string{fmt.Sprintf("[%s] INFO  - Primary network controller initialization handshake complete.", time.Now().Format("15:04:05"))},
		nodeState:  make(map[string]string),
		tickRate:   time.Second,
		prog:       prog,
		textInput:  ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		fetchMetrics(m.cfg, m.extendedDNS),
		func() tea.Msg {
			meta, err := diagnostics.FetchWANMetadata()
			return wanMsg{meta: meta, err: err}
		},
		textinput.Blink,
	)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.tickRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchMetrics(cfg *config.Config, extendedDNS bool) tea.Cmd {
	return func() tea.Msg {
		gwIP, err := diagnostics.GetDefaultGateway()
		if err != nil {
			gwIP = "1.1.1.1"
		}

		gwRes := make(chan diagnostics.PingResult, 1)
		go func() {
			gwRes <- diagnostics.Ping(gwIP)
		}()

		bbRes := make(chan []diagnostics.PingResult, 1)
		go func() {
			var res []diagnostics.PingResult
			for _, t := range cfg.Targets.Backbones {
				res = append(res, diagnostics.Ping(t))
			}
			bbRes <- res
		}()

		dnsRes := make(chan []diagnostics.DNSResult, 1)
		go func() {
			var res []diagnostics.DNSResult
			for _, p := range cfg.Targets.DNS {
				res = append(res, diagnostics.ResolveDNS(p.Name, p.Addr, "google.com"))
			}
			dnsRes <- res
		}()

		ifaceRes := make(chan []diagnostics.IfaceStats, 1)
		go func() {
			stats, _ := diagnostics.GetIfaceStats()
			ifaceRes <- stats
		}()

		return metricsMsg{
			gwPing:    <-gwRes,
			backbones: <-bbRes,
			dns:       <-dnsRes,
			ifaces:    <-ifaceRes,
		}
	}
}

func runTraceroute(target string) tea.Cmd {
	return func() tea.Msg {
		hops, err := diagnostics.Traceroute(target, 30)
		return tracerouteMsg{hops: hops, err: err}
	}
}

func appendHistory(history []float64, val float64) []float64 {
	history = append(history, val)
	if len(history) > 500 {
		history = history[len(history)-500:]
	}
	return history
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok && m.footerError != "" {
		m.footerError = ""
	}

	// 1. Process asynchronous non-key messages first to prevent ticker freezing!
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.width = int(float64(msg.Width) * 0.95)
		m.height = msg.Height
		return m, nil
	case tickMsg:
		var cmd tea.Cmd
		if !m.paused && !m.mtrMode {
			cmd = tea.Batch(m.tickCmd(), fetchMetrics(m.cfg, m.extendedDNS))
		} else {
			cmd = m.tickCmd()
		}

		// If input is active, we MUST pass the tick to textInput but also return our tickCmd!
		if m.inputActive {
			var tiCmd tea.Cmd
			m.textInput, tiCmd = m.textInput.Update(msg)
			return m, tea.Batch(cmd, tiCmd)
		}
		return m, cmd
	case wanMsg:
		m.wanMeta = msg.meta
		if msg.err != nil {
			m.footerError = "WAN Metadata Error: " + msg.err.Error()
		} else if msg.meta != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] BGP   - Geo-node localization complete: %s, %s [%s].", time.Now().Format("15:04:05"), msg.meta.City, msg.meta.Region, msg.meta.Org))
		}
		return m, nil
	case metricsMsg:
		if !m.paused && !m.mtrMode {
			m.gwPing = msg.gwPing
			m.backbones = msg.backbones

			// Update history
			m.gwHistory = appendHistory(m.gwHistory, float64(msg.gwPing.Latency.Microseconds())/1000.0)

			m.dns = msg.dns
			for _, d := range m.dns {
				val := float64(d.Latency.Microseconds()) / 1000.0
				if d.Error != nil {
					val = -1
				}
				m.dnsHistory[d.Provider] = appendHistory(m.dnsHistory[d.Provider], val)

				oldState := m.nodeState[d.Provider]
				newState := "ONLINE"
				if d.Error != nil {
					newState = "FAULT"
				} else if val > float64(m.cfg.Daemon.AlertThreshold) {
					newState = "WARN"
				}
				if oldState != "" && oldState != newState {
					if newState == "FAULT" {
						m.logs = append(m.logs, fmt.Sprintf("[%s] ALERT - DNS %s resolution timeout fault detected.", time.Now().Format("15:04:05"), d.Provider))
					} else if newState == "WARN" {
						m.logs = append(m.logs, fmt.Sprintf("[%s] ALERT - DNS %s latency exceeded threshold limit (>%.0fms).", time.Now().Format("15:04:05"), d.Provider, float64(m.cfg.Daemon.AlertThreshold)))
					} else if newState == "ONLINE" {
						m.logs = append(m.logs, fmt.Sprintf("[%s] INFO  - DNS %s has recovered and is fully operational.", time.Now().Format("15:04:05"), d.Provider))
					}
				}
				m.nodeState[d.Provider] = newState
			}

			for _, b := range m.backbones {
				val := float64(b.Latency.Microseconds()) / 1000.0
				if b.Error != nil || b.Loss > 0 {
					val = -1
				}
				m.bbHistory[b.Target] = appendHistory(m.bbHistory[b.Target], val)

				oldState := m.nodeState[b.Target]
				newState := "ONLINE"
				if b.Error != nil || b.Loss > 0 {
					newState = "FAULT"
				} else if val > float64(m.cfg.Daemon.AlertThreshold) {
					newState = "WARN"
				}
				if oldState != "" && oldState != newState {
					if newState == "FAULT" {
						m.logs = append(m.logs, fmt.Sprintf("[%s] ALERT - Node %s timeout fault detected.", time.Now().Format("15:04:05"), b.Target))
					} else if newState == "WARN" {
						m.logs = append(m.logs, fmt.Sprintf("[%s] ALERT - Node %s latency exceeded threshold limit (>%.0fms).", time.Now().Format("15:04:05"), b.Target, float64(m.cfg.Daemon.AlertThreshold)))
					} else if newState == "ONLINE" {
						m.logs = append(m.logs, fmt.Sprintf("[%s] INFO  - Node %s has recovered and is fully operational.", time.Now().Format("15:04:05"), b.Target))
					}
				}
				m.nodeState[b.Target] = newState
			}

			if len(msg.ifaces) > 0 {
				m.ifaces = msg.ifaces
				m.rxHistory = appendHistory(m.rxHistory, float64(msg.ifaces[0].RXSpeed))
				m.txHistory = appendHistory(m.txHistory, float64(msg.ifaces[0].TXSpeed))
			}

			if len(m.logs) > 100 {
				m.logs = m.logs[len(m.logs)-100:]
			}
		}
		if m.inputActive {
			var tiCmd tea.Cmd
			m.textInput, tiCmd = m.textInput.Update(msg)
			return m, tiCmd
		}
		return m, nil
	case tracerouteMsg:
		m.mtrLoading = false
		m.mtrHops = msg.hops
		m.mtrErr = msg.err
		if m.inputActive {
			var tiCmd tea.Cmd
			m.textInput, tiCmd = m.textInput.Update(msg)
			return m, tiCmd
		}
		return m, nil
	}

	// 2. If input is active, intercept all key presses exclusively
	if m.inputActive {
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				val := m.textInput.Value()
				if m.renameActive {
					if val != "" && m.focus == FocusDNS && m.dnsCursor < len(m.cfg.Targets.DNS) {
						m.cfg.Targets.DNS[m.dnsCursor].Name = val
						config.SaveConfig(m.cfg)
					}
					m.renameActive = false
				} else if val != "" {
					if m.focus == FocusDNS {
						name := "Custom"
						addr := val
						if strings.Contains(val, ",") {
							parts := strings.SplitN(val, ",", 2)
							name = strings.TrimSpace(parts[0])
							addr = strings.TrimSpace(parts[1])
						} else if strings.Contains(val, " ") {
							parts := strings.SplitN(val, " ", 2)
							name = strings.TrimSpace(parts[0])
							addr = strings.TrimSpace(parts[1])
						}
						m.cfg.Targets.DNS = append(m.cfg.Targets.DNS, struct {
							Name string `toml:"name"`
							Addr string `toml:"addr"`
						}{Name: name, Addr: addr})
						m.logs = append(m.logs, fmt.Sprintf("[%s] USER  - Dynamic upstream target registered by node profile user: %s", time.Now().Format("15:04:05"), name))
					} else if m.focus == FocusBackbones {
						m.cfg.Targets.Backbones = append(m.cfg.Targets.Backbones, val)
						m.logs = append(m.logs, fmt.Sprintf("[%s] USER  - Dynamic upstream target registered by node profile user: %s", time.Now().Format("15:04:05"), val))
					}
					config.SaveConfig(m.cfg)
				}
				m.inputActive = false
				m.textInput.SetValue("")
				m.textInput.Blur()
				return m, fetchMetrics(m.cfg, m.extendedDNS)
			case "esc":
				m.inputActive = false
				m.renameActive = false
				m.textInput.SetValue("")
				m.textInput.Blur()
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// 3. Normal layout keystrokes
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.mtrMode {
				m.mtrMode = false
			} else {
				m.paused = !m.paused
			}
			return m, nil
		case "tab", "right":
			if !m.mtrMode {
				m.focus = (m.focus + 1) % 5
			}
			return m, nil
		case "shift+tab", "left":
			if !m.mtrMode {
				m.focus--
				if m.focus < 0 {
					m.focus = FocusBackbones
				}
			}
			return m, nil
		case "j", "down":
			if !m.mtrMode {
				if m.focus == FocusDNS {
					m.dnsCursor++
					if m.dnsCursor >= len(m.cfg.Targets.DNS) {
						m.dnsCursor = 0
					}
				} else if m.focus == FocusBackbones {
					m.bbCursor++
					if m.bbCursor >= len(m.cfg.Targets.Backbones) {
						m.bbCursor = 0
					}
				}
			}
			return m, nil
		case "k", "up":
			if !m.mtrMode {
				if m.focus == FocusDNS {
					m.dnsCursor--
					if m.dnsCursor < 0 {
						m.dnsCursor = len(m.cfg.Targets.DNS) - 1
					}
				} else if m.focus == FocusBackbones {
					m.bbCursor--
					if m.bbCursor < 0 {
						m.bbCursor = len(m.cfg.Targets.Backbones) - 1
					}
				}
			}
			return m, nil
		case "x", "delete", "backspace":
			if !m.mtrMode {
				if m.focus == FocusDNS && len(m.cfg.Targets.DNS) > 0 {
					m.cfg.Targets.DNS = append(m.cfg.Targets.DNS[:m.dnsCursor], m.cfg.Targets.DNS[m.dnsCursor+1:]...)
					if m.dnsCursor >= len(m.cfg.Targets.DNS) {
						m.dnsCursor = len(m.cfg.Targets.DNS) - 1
					}
					if m.dnsCursor < 0 {
						m.dnsCursor = 0
					}
					config.SaveConfig(m.cfg)
					return m, fetchMetrics(m.cfg, m.extendedDNS)
				} else if m.focus == FocusBackbones && len(m.cfg.Targets.Backbones) > 0 {
					m.cfg.Targets.Backbones = append(m.cfg.Targets.Backbones[:m.bbCursor], m.cfg.Targets.Backbones[m.bbCursor+1:]...)
					if m.bbCursor >= len(m.cfg.Targets.Backbones) {
						m.bbCursor = len(m.cfg.Targets.Backbones) - 1
					}
					if m.bbCursor < 0 {
						m.bbCursor = 0
					}
					config.SaveConfig(m.cfg)
					return m, fetchMetrics(m.cfg, m.extendedDNS)
				}
			}
			return m, nil
		case "a", "A":
			if !m.mtrMode {
				if m.focus == FocusDNS {
					m.inputActive = true
					m.textInput.Placeholder = "ENTER UPSTREAM DNS SERVER IP (e.g., 9.9.9.9): "
					m.textInput.Focus()
					return m, textinput.Blink
				} else if m.focus == FocusBackbones {
					m.inputActive = true
					m.textInput.Placeholder = "ENTER TARGET WAN DOMAIN / IP (e.g., github.com): "
					m.textInput.Focus()
					return m, textinput.Blink
				} else {
					m.footerError = "SYSTEM NOTICE: Hardware, Gateway and WAN profiles are read-only."
					return m, nil
				}
			}
		case "enter", "t", "T":
			if m.focus == FocusBackbones && !m.mtrMode {
				m.mtrMode = true
				if !m.mtrLoading {
					m.mtrLoading = true
					target := "8.8.8.8"
					if len(m.cfg.Targets.Backbones) > 0 && m.bbCursor < len(m.cfg.Targets.Backbones) {
						target = m.cfg.Targets.Backbones[m.bbCursor]
					}
					return m, runTraceroute(target)
				}
			}
			return m, nil
		case "+", "=":
			if m.tickRate > 250*time.Millisecond {
				m.tickRate /= 2
				if m.tickRate < 250*time.Millisecond {
					m.tickRate = 250 * time.Millisecond
				}
			}
			return m, nil
		case "-":
			if m.tickRate < 5*time.Second {
				m.tickRate *= 2
				if m.tickRate > 5*time.Second {
					m.tickRate = 5 * time.Second
				}
			}
			return m, nil
		case " ":
			if !m.paused && !m.mtrMode {
				return m, fetchMetrics(m.cfg, m.extendedDNS)
			}
		case "r", "R":
			if !m.mtrMode && m.focus == FocusDNS && len(m.cfg.Targets.DNS) > 0 {
				m.renameActive = true
				m.inputActive = true
				m.textInput.Placeholder = "Enter new name..."
				m.textInput.SetValue(m.cfg.Targets.DNS[m.dnsCursor].Name)
				m.textInput.Focus()
				return m, textinput.Blink
			}
			if !m.paused && !m.mtrMode && m.focus != FocusDNS {
				return m, fetchMetrics(m.cfg, m.extendedDNS)
			}
		case "d", "D":
			m.extendedDNS = !m.extendedDNS
			if !m.paused && !m.mtrMode {
				return m, fetchMetrics(m.cfg, m.extendedDNS)
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing UI..."
	}

	var ui string

	if m.mtrMode {
		ui = RenderMTR(m)
	} else {
		nodeName, _ := os.Hostname()
		if nodeName == "" {
			nodeName = "unknown"
		}
		if m.isDemo {
			nodeName = "demo_node"
		}

		headerText := fmt.Sprintf("NetPulse Network Telemetry // Node: %s // Local Time: %v", nodeName, m.tickRate)
		if m.paused {
			headerText += " [PAUSED]"
		}

		header := lipgloss.NewStyle().
			Bold(true).
			Foreground(NeonPurple).
			MarginBottom(1).
			Width(m.width).
			Render(headerText)

		// Exact calculation of extra vertical lines (borders + margins)
		overhead := 12
		available := m.height - overhead
		if available < 10 {
			available = 10 // Prevent panic on microscopic terminals
		}

		row1Height := 8
		if len(m.dns) > 4 {
			row1Height = len(m.dns) + 4
		}
		// Prevent DNS list from consuming the entire screen
		if row1Height > available/2 {
			row1Height = available / 2
		}
		if row1Height < 8 {
			row1Height = 8
		}

		row2Height := 9
		if row2Height > available/3 {
			row2Height = available / 3
		}
		if row2Height < 9 {
			row2Height = 9
		}

		bbHeight := len(m.backbones) + 5
		logHeight := available - (row1Height + row2Height + bbHeight)

		// Balance the bottom panels
		if logHeight < 5 {
			logHeight = 5
			bbHeight = available - (row1Height + row2Height + logHeight)
			if bbHeight < 4 {
				bbHeight = 4
			}
		}

		gwWidth := int(float64(m.width) * 0.40)
		dnsWidth := m.width - gwWidth

		ifaceWidth := int(float64(m.width) * 0.60)
		wanWidth := m.width - ifaceWidth

		gw := RenderGateway(m, gwWidth, row1Height)
		dns := RenderDNS(m, dnsWidth, row1Height)
		iface := RenderIface(m, ifaceWidth, row2Height)
		wan := RenderWAN(m, wanWidth, row2Height)
		bb := RenderBackbones(m, bbHeight)
		logs := RenderLogs(m, logHeight)

		row1 := lipgloss.JoinHorizontal(lipgloss.Top, gw, dns)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, iface, wan)
		middle := lipgloss.JoinVertical(lipgloss.Left, row1, row2)

		footerText := " 🕹️  "
		if m.focus == FocusDNS || m.focus == FocusBackbones {
			footerText += "[Tab] Cycle Modules • [j/k] Move • [A] Add • [x] Delete"
			if m.focus == FocusDNS {
				footerText += " • [r] Rename"
			}
			if m.focus == FocusBackbones {
				footerText += " • [T] Enter MTR"
			}
		} else {
			footerText += "[Tab] Cycle Modules • [A] Add • [+/-] Tick"
		}
		footerText += " • [Esc] Pause • [Q] Abort"

		if m.footerError != "" {
			footerText = lipgloss.NewStyle().Foreground(GlitchRed).Bold(true).Render(" ▲ " + m.footerError)
		} else if m.inputActive {
			prompt := " [ Add Custom Node ] ► "
			if m.renameActive {
				prompt = " [ Rename DNS Node ] ► "
			}
			footerText = lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Foreground(NeonPurple).Bold(true).Render(prompt),
				m.textInput.View(),
				lipgloss.NewStyle().Foreground(MatrixGray).Render(" (Press Enter to save, Esc to cancel)"),
			)
		}
		footer := lipgloss.NewStyle().Foreground(SubduedCyan).MarginTop(1).Render(footerText)

		ui = lipgloss.JoinVertical(lipgloss.Left, header, middle, bb, logs, footer)
	}

	marginLeft := (m.termWidth - m.width) / 2
	return lipgloss.NewStyle().MarginLeft(marginLeft).Render(ui)
}
