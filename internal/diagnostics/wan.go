package diagnostics

import (
	"encoding/json"
	"net/http"
	"time"
)

type WANMetadata struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Org     string `json:"org"` // ISP/ASN
}

var cachedWAN *WANMetadata

func FetchWANMetadata() (*WANMetadata, error) {
	if cachedWAN != nil {
		return cachedWAN, nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ipinfo.io/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var wan WANMetadata
	if err := json.NewDecoder(resp.Body).Decode(&wan); err != nil {
		return nil, err
	}

	cachedWAN = &wan
	return cachedWAN, nil
}
