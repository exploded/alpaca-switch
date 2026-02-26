package server

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// StartDiscovery listens for ASCOM Alpaca UDP discovery broadcasts on listenPort
// and responds with the given apiPort.
//
// NINA sends a discovery packet from every local network interface simultaneously,
// which can cause duplicate listings. We reduce this by:
//  1. Determining the machine's primary LAN IP (via outboundIP).
//  2. Responding to loopback packets (for same-host clients) and packets on the
//     same /24 subnet as the LAN IP.
//  3. Deduplicating responses per source IP within a 2-second window.
func StartDiscovery(listenPort, apiPort int) {
	addr := fmt.Sprintf("0.0.0.0:%d", listenPort)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("Discovery listener failed to bind on %s: %v", addr, err)
	}
	defer conn.Close()

	lanIP := outboundIP()
	log.Printf("Discovery listener binding to %s (LAN IP: %s)", addr, lanIP)
	reply := fmt.Sprintf("{\n\"AlpacaPort\":%d\n}", apiPort)

	// recentReplies deduplicates within a 2-second window as a safety net.
	var mu sync.Mutex
	recentReplies := make(map[string]time.Time)

	buf := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("Discovery read error: %v", err)
			continue
		}
		msg := string(buf[:n])
		if !strings.HasPrefix(strings.TrimSpace(msg), "alpacadiscovery1") {
			continue
		}

		srcUDP := src.(*net.UDPAddr)
		srcIP := srcUDP.IP.String()

		// Respond to local loopback packets for same-host clients (e.g. NINA),
		// and to packets from the same /24 subnet as our LAN IP.
		if !isLoopbackIP(srcUDP.IP) && !sameSubnet24(srcIP, lanIP) {
			log.Printf("Discovery: ignoring packet from %s (not on LAN subnet %s/24)", srcIP, lanIP)
			continue
		}

		// Deduplicate: only reply once per source IP per 2 seconds.
		mu.Lock()
		last, seen := recentReplies[srcIP]
		if seen && time.Since(last) < 2*time.Second {
			mu.Unlock()
			continue
		}
		recentReplies[srcIP] = time.Now()
		mu.Unlock()

		log.Printf("Received discovery packet from %s, sending response", src)
		if _, err := conn.WriteTo([]byte(reply), src); err != nil {
			log.Printf("Discovery response error: %v", err)
		}
	}
}

// sameSubnet24 returns true if ip and ref share the same first three octets.
func sameSubnet24(ip, ref string) bool {
	a := net.ParseIP(ip).To4()
	b := net.ParseIP(ref).To4()
	if a == nil || b == nil {
		return false
	}
	return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
}

func isLoopbackIP(ip net.IP) bool {
	return ip != nil && ip.IsLoopback()
}

// outboundIP returns the IP of the interface used for outbound traffic.
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("outboundIP: could not determine local IP, falling back to 0.0.0.0: %v", err)
		return "0.0.0.0"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
