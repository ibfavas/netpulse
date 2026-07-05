package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ibfavas/netpulse/internal/config"
	"github.com/ibfavas/netpulse/internal/daemon"
	"github.com/ibfavas/netpulse/internal/ui"
)

func main() {
	daemonMode := flag.Bool("daemon", false, "Run in background daemon mode")
	demoMode := flag.Bool("demo", false, "Run in demo mode with mocked data for clean screenshots")
	flag.Parse()

	cfg := config.LoadConfig()

	if *demoMode {
		cfg.Targets.DNS[0].Name = "demo_user"
	}

	if *daemonMode {
		daemon.Run(cfg)
		return
	}

	p := tea.NewProgram(ui.InitialModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
