package diagnostics

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type PingResult struct {
	Target  string
	Latency time.Duration
	Loss    float64
	Error   error
}

func Ping(target string) PingResult {
	addr, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return PingResult{Target: target, Loss: 100, Error: err}
	}

	// Try listening on icmp (requires root/CAP_NET_RAW)
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		// Fallback to udp ping if available (unprivileged ping)
		conn, err = icmp.ListenPacket("udp4", "0.0.0.0")
		if err != nil {
			return PingResult{Target: target, Loss: 100, Error: err}
		}
	}
	defer conn.Close()

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("HELLO-NETPULSE"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return PingResult{Target: target, Loss: 100, Error: err}
	}

	start := time.Now()

	var dst net.Addr = addr
	if conn.LocalAddr().Network() == "udp4" {
		dst = &net.UDPAddr{IP: addr.IP, Zone: addr.Zone}
	}

	if _, err := conn.WriteTo(wb, dst); err != nil {
		return PingResult{Target: target, Loss: 100, Error: err}
	}

	// Read reply
	err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return PingResult{Target: target, Loss: 100, Error: err}
	}

	rb := make([]byte, 1500)
	n, _, err := conn.ReadFrom(rb)
	if err != nil {
		return PingResult{Target: target, Loss: 100, Error: err} // timeout means loss
	}

	latency := time.Since(start)

	rm, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		return PingResult{Target: target, Loss: 100, Error: err}
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return PingResult{Target: target, Latency: latency, Loss: 0}
	default:
		return PingResult{Target: target, Loss: 100, Error: fmt.Errorf("unexpected icmp type: %v", rm.Type)}
	}
}

func GetDefaultGateway() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == "00000000" {
			ipHex := fields[2]
			if len(ipHex) == 8 {
				var b1, b2, b3, b4 byte
				fmt.Sscanf(ipHex, "%02x%02x%02x%02x", &b4, &b3, &b2, &b1)
				ip := net.IPv4(b1, b2, b3, b4)
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("default gateway not found")
}
