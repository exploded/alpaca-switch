// mi-switch is a minimal CLI for controlling Xiaomi Mi Smart Plugs directly,
// without launching the ASCOM Alpaca server.
//
// Usage:
//   mi-switch --host <IP> --token <32-hex-token> --action on|off|status
package main

import (
	"flag"
	"fmt"
	"os"

	"alpaca-switch/backend/mi"
)

func main() {
	host := flag.String("host", "", "Device IP address (required)")
	token := flag.String("token", "", "32-character hex token (required)")
	action := flag.String("action", "", "Action: on | off | status (required)")
	flag.Parse()

	if *host == "" || *token == "" || *action == "" {
		fmt.Fprintln(os.Stderr, "Usage: mi-switch --host <IP> --token <32-hex-token> --action on|off|status")
		os.Exit(1)
	}

	switch *action {
	case "on":
		if err := mi.SetSwitch(*host, *token, true); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Switch: on")

	case "off":
		if err := mi.SetSwitch(*host, *token, false); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Switch: off")

	case "status":
		on, err := mi.GetSwitch(*host, *token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if on {
			fmt.Println("Switch: on")
		} else {
			fmt.Println("Switch: off")
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown action %q -- must be on, off, or status\n", *action)
		os.Exit(1)
	}
}
