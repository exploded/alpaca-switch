package server

import (
	"fmt"
	"log"
	"net"
	"strings"
)

// StartDiscovery listens for ASCOM Alpaca UDP discovery broadcasts on listenPort
// and responds with the given apiPort. Binds to 0.0.0.0 so it receives broadcasts.
func StartDiscovery(listenPort, apiPort int) {
	addr := fmt.Sprintf("0.0.0.0:%d", listenPort)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("Discovery listener failed to bind on %s: %v", addr, err)
	}
	defer conn.Close()

	log.Printf("Discovery listener binding to %s", addr)
	reply := fmt.Sprintf(`{"AlpacaPort":%d}`, apiPort)

	buf := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("Discovery read error: %v", err)
			continue
		}
		msg := string(buf[:n])
		log.Printf("Received discovery packet from %s", src)
		if strings.HasPrefix(strings.TrimSpace(msg), "alpacadiscovery1") {
			log.Printf("Sending discovery response to %s", src)
			if _, err := conn.WriteTo([]byte(reply), src); err != nil {
				log.Printf("Discovery response error: %v", err)
			}
		}
	}
}
