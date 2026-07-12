package network

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type InterfaceInfo struct {
	Name          string   `json:"name"`
	MAC           string   `json:"mac"`
	Status        string   `json:"status"`
	MTU           int      `json:"mtu"`
	IPv4          []string `json:"ipv4,omitempty"`
	IPv6          []string `json:"ipv6,omitempty"`
	Gateway       string   `json:"gateway,omitempty"`
	BytesSent     uint64   `json:"bytes_sent"`
	BytesRecv     uint64   `json:"bytes_recv"`
	PacketsSent   uint64   `json:"packets_sent"`
	PacketsRecv   uint64   `json:"packets_recv"`
	ErrorsIn      uint64   `json:"errors_in"`
	ErrorsOut     uint64   `json:"errors_out"`
	SpeedMbps     float64  `json:"speed_mbps"`
	SpeedRecvMbps float64  `json:"speed_recv_mbps,omitempty"`
	SpeedSentMbps float64  `json:"speed_sent_mbps,omitempty"`
	Duplex        string   `json:"duplex,omitempty"`
}

type Info struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
	DNSServers []string        `json:"dns_servers,omitempty"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
