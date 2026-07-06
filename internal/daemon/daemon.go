package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/ibfavas/netpulse/internal/config"
	"github.com/ibfavas/netpulse/internal/diagnostics"
)

func SendNotification(title, body string) {
	conn, err := dbus.SessionBus()
	if err == nil {
		obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
		call := obj.Call("org.freedesktop.Notifications.Notify", 0,
			"NetPulse", uint32(0), "", title, body, []string{}, map[string]dbus.Variant{}, int32(5000))
		if call.Err == nil {
			return
		}
	}

	// Fallback for sudo execution
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		uidCmd := exec.Command("id", "-u", sudoUser)
		uidOut, err := uidCmd.Output()
		if err == nil {
			uid := strings.TrimSpace(string(uidOut))
			dbusAddr := fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%s/bus", uid)
			cmd := exec.Command("sudo", "-u", sudoUser, "env", dbusAddr, "notify-send", title, body)
			_ = cmd.Run()
		}
	}
}

type LogEntry struct {
	Timestamp string  `json:"timestamp"`
	Gateway   float64 `json:"gateway_ms"`
	Outage    bool    `json:"outage"`
}

func Run(cfg *config.Config) {
	fmt.Println("NetPulse Daemon started in background...")
	fmt.Printf("Logging to: %s\n", cfg.Daemon.LogFile)
	fmt.Println("Press Ctrl+C to stop.")

	logPath := cfg.Daemon.LogFile
	if strings.HasPrefix(logPath, "~/") {
		home, _ := os.UserHomeDir()
		logPath = filepath.Join(home, logPath[2:])
	}
	os.MkdirAll(filepath.Dir(logPath), 0755)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer f.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.Daemon.Interval) * time.Second)
	defer ticker.Stop()

	wasDown := false

	// Initial notification check
	SendNotification("NetPulse", "Daemon started monitoring your connection.")

	nodeState := make(map[string]string)

	for {
		select {
		case <-sigChan:
			fmt.Println("\nDaemon shutting down.")
			return
		case <-ticker.C:
			// Ping Gateway
			gwIP, _ := diagnostics.GetDefaultGateway()
			if gwIP == "" {
				gwIP = "1.1.1.1"
			}
			res := diagnostics.Ping(gwIP)

			ms := float64(res.Latency.Microseconds()) / 1000.0
			isDown := res.Loss > 0 || res.Error != nil

			entry := LogEntry{
				Timestamp: time.Now().Format(time.RFC3339),
				Gateway:   ms,
				Outage:    isDown,
			}
			b, _ := json.Marshal(entry)
			f.WriteString(string(b) + "\n")

			if isDown && !wasDown {
				SendNotification("NetPulse Alert", "Gateway connection lost!")
				wasDown = true
			} else if !isDown && wasDown {
				SendNotification("NetPulse Alert", "Gateway connection restored.")
				wasDown = false
			}

			// Monitor Targets (Backbones + DNS)
			for _, bb := range cfg.Targets.Backbones {
				checkTargetState(bb, bb, float64(cfg.Daemon.AlertThreshold), nodeState)
			}
			for _, dns := range cfg.Targets.DNS {
				checkTargetState(dns.Addr, dns.Name, float64(cfg.Daemon.AlertThreshold), nodeState)
			}
		}
	}
}

func checkTargetState(ip, name string, threshold float64, nodeState map[string]string) {
	res := diagnostics.Ping(ip)
	ms := float64(res.Latency.Microseconds()) / 1000.0

	newState := "ONLINE"
	if res.Loss > 0 || res.Error != nil {
		newState = "FAULT"
	} else if ms > threshold {
		newState = "WARN"
	}

	oldState := nodeState[name]
	if oldState != "" && oldState != newState {
		if newState == "FAULT" {
			SendNotification("NetPulse Alert", fmt.Sprintf("Node %s timeout fault detected.", name))
		} else if newState == "WARN" {
			SendNotification("NetPulse Alert", fmt.Sprintf("Node %s latency exceeded threshold (>%.0fms).", name, threshold))
		} else if newState == "ONLINE" {
			SendNotification("NetPulse Info", fmt.Sprintf("Node %s has recovered and is fully operational.", name))
		}
	}
	nodeState[name] = newState
}
