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
// It binds to 0.0.0.0 so it receives broadcasts on all interfaces, but
// deduplicates responses so each discovering client (identified by source IP)
// only receives one reply per 2-second window. This prevents NINA from listing
// the driver multiple times when the machine has several network adapters
// (e.g. LAN, WSL virtual adapter, loopback).
func StartDiscovery(listenPort, apiPort int) {
	addr := fmt.Sprintf("0.0.0.0:%d", listenPort)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("Discovery listener failed to bind on %s: %v", addr, err)
	}
	defer conn.Close()

	log.Printf("Discovery listener binding to %s", addr)
	reply := fmt.Sprintf("{\n\"AlpacaPort\":%d\n}", apiPort)

	// recentReplies tracks source IPs we have already replied to recently.
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

		// Extract just the IP (strip port) for deduplication key.
		srcIP := src.(*net.UDPAddr).IP.String()

		mu.Lock()
		last, seen := recentReplies[srcIP]
		if seen && time.Since(last) < 2*time.Second {
			mu.Unlock()
			log.Printf("Discovery: suppressing duplicate reply to %s", srcIP)
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
