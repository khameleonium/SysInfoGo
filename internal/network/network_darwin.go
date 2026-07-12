package network

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	interfaces, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("network interfaces: %w", err)
	}

	counters, _ := net.IOCountersWithContext(ctx, true)

	var result []InterfaceInfo
	for _, iface := range interfaces {
		ni := InterfaceInfo{
			Name:   iface.Name,
			MAC:    iface.HardwareAddr,
			MTU:    iface.MTU,
			Status: "down",
		}
		for _, f := range iface.Flags {
			if strings.ToLower(f) == "up" {
				ni.Status = "up"
				break
			}
		}
		for _, addr := range iface.Addrs {
			if strings.Contains(addr.Addr, ":") {
				ni.IPv6 = append(ni.IPv6, addr.Addr)
			} else {
				ni.IPv4 = append(ni.IPv4, addr.Addr)
			}
		}

		for _, c := range counters {
			if c.Name == iface.Name {
				ni.BytesSent = c.BytesSent
				ni.BytesRecv = c.BytesRecv
				ni.PacketsSent = c.PacketsSent
				ni.PacketsRecv = c.PacketsRecv
				ni.ErrorsIn = c.Errin
				ni.ErrorsOut = c.Errout
				break
			}
		}

		result = append(result, ni)
	}

	dns := collectDNSFile()
	return &Info{Interfaces: result, DNSServers: dns}, nil, nil
}

func collectDNSFile() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}

	var dns []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				dns = append(dns, parts[1])
			}
		}
	}
	return dns
}
