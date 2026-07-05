package diagnostics

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type IfaceStats struct {
	Name      string
	IP        string
	MAC       string
	RXBytes   uint64
	TXBytes   uint64
	RXSpeed   float64 // bytes per second
	TXSpeed   float64
	Timestamp time.Time
}

var lastStats map[string]IfaceStats

func GetIfaceStats() ([]IfaceStats, error) {
	if lastStats == nil {
		lastStats = make(map[string]IfaceStats)
	}

	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ifaceMap := make(map[string]net.Interface)
	for _, iface := range ifaces {
		ifaceMap[iface.Name] = iface
	}

	var results []IfaceStats
	now := time.Now()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Split(line, ":")
		name := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		if name == "lo" {
			continue
		}

		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)

		stat := IfaceStats{
			Name:      name,
			RXBytes:   rx,
			TXBytes:   tx,
			Timestamp: now,
		}

		if iface, ok := ifaceMap[name]; ok {
			stat.MAC = iface.HardwareAddr.String()
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						stat.IP = ipnet.IP.String()
						break
					}
				}
			}
		}

		if last, ok := lastStats[name]; ok {
			duration := now.Sub(last.Timestamp).Seconds()
			if duration > 0 {
				stat.RXSpeed = float64(rx-last.RXBytes) / duration
				stat.TXSpeed = float64(tx-last.TXBytes) / duration
			}
		}

		lastStats[name] = stat
		results = append(results, stat)
	}
	return results, nil
}
