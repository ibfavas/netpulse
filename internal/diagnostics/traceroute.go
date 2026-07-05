package diagnostics

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Hop struct {
	TTL     int
	IP      string
	Latency time.Duration
	Lost    bool
}

func Traceroute(target string, maxHops int) ([]Hop, error) {
	addr, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return nil, err
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, fmt.Errorf("MTR mode requires raw socket permissions. Try running with 'sudo' or 'setcap cap_net_raw+ep'")
	}
	defer conn.Close()

	pconn := conn.IPv4PacketConn()

	var hops []Hop
	for ttl := 1; ttl <= maxHops; ttl++ {
		pconn.SetTTL(ttl)

		wm := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  ttl,
				Data: []byte("NETPULSE-MTR"),
			},
		}
		wb, err := wm.Marshal(nil)
		if err != nil {
			continue
		}

		start := time.Now()
		if _, err := conn.WriteTo(wb, addr); err != nil {
			hops = append(hops, Hop{TTL: ttl, Lost: true})
			continue
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		rb := make([]byte, 1500)
		n, peer, err := conn.ReadFrom(rb)
		latency := time.Since(start)

		if err != nil {
			hops = append(hops, Hop{TTL: ttl, Lost: true})
			continue
		}

		rm, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			hops = append(hops, Hop{TTL: ttl, Lost: true})
			continue
		}

		var peerIP string
		if peer != nil {
			peerIP = peer.String()
		} else {
			peerIP = "*.*.*.*"
		}

		if rm.Type == ipv4.ICMPTypeTimeExceeded {
			hops = append(hops, Hop{TTL: ttl, IP: peerIP, Latency: latency})
		} else if rm.Type == ipv4.ICMPTypeEchoReply {
			hops = append(hops, Hop{TTL: ttl, IP: peerIP, Latency: latency})
			break // Reached destination
		} else {
			hops = append(hops, Hop{TTL: ttl, IP: peerIP, Latency: latency})
		}
	}

	return hops, nil
}
