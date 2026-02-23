package server

import (
	"fmt"
	"log"
	"net"
	"strings"
)

// StartDiscovery listens for ASCOM Alpaca UDP discovery broadcasts on listenPort
// and responds with the given apiPort.
//
// It binds to the primary outbound network interface (not 0.0.0.0) so that NINA
// only discovers this driver once, even on machines with multiple virtual or
// physical network adapters.
func StartDiscovery(listenPort, apiPort int) {
	listenIP := outboundIP()
	addr := fmt.Sprintf("%s:%d", listenIP, listenPort)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("Discovery listener failed to bind on %s: %v", addr, err)
	}
	defer conn.Close()

	log.Printf("Discovery listener binding to %s", addr)
	reply := fmt.Sprintf("{\n\"AlpacaPort\":%d\n}", apiPort)

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

// outboundIP returns the IP of the interface used for outbound traffic.
// This avoids binding to loopback or virtual adapter addresses, which would
// cause NINA to discover the driver multiple times.
func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("outboundIP: could not determine local IP, falling back to 0.0.0.0: %v", err)
		return "0.0.0.0"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
