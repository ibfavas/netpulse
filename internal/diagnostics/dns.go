package diagnostics

import (
	"context"
	"net"
	"time"
)

type DNSResult struct {
	Provider string
	Latency  time.Duration
	Error    error
}

func ResolveDNS(providerName, serverAddr, domain string) DNSResult {
	start := time.Now()

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(1000),
			}
			if serverAddr == "system" {
				return d.DialContext(ctx, network, address)
			}
			return d.DialContext(ctx, "udp", serverAddr+":53")
		},
	}

	_, err := r.LookupHost(context.Background(), domain)
	latency := time.Since(start)

	return DNSResult{
		Provider: providerName,
		Latency:  latency,
		Error:    err,
	}
}
