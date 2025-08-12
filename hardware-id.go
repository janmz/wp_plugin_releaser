package main

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

// getHardwareID generates a unique ID based on hardware properties
func getHardwareID() (uint64, error) {
	var identifiers []string

	// MAC address of the first network card
	interfaces, err := net.Interfaces()
	if err == nil && len(interfaces) > 0 {
		for _, iface := range interfaces {
			if iface.HardwareAddr != nil {
				identifiers = append(identifiers, iface.HardwareAddr.String())
				break
			}
		}
	}

	// CPU ID and other hardware information depending on the operating system
	switch runtime.GOOS {
	case "windows":
		// Windows-specific hardware IDs
		cmds := []string{
			"wmic cpu get ProcessorId",
			"wmic baseboard get SerialNumber",
			"wmic baseboard get Product",
			"wmic diskdrive get SerialNumber",
		}

		for _, cmd := range cmds {
			out, err := exec.Command("cmd", "/C", cmd).Output()
			if err == nil {
				lines := strings.Split(string(out), "\n")
				if len(lines) > 1 {
					// First line is the header, second line contains the value
					value := strings.TrimSpace(lines[1])
					if value != "" {
						identifiers = append(identifiers, value)
					}
				}
			}
		}

	case "linux":
		// Linux-specific hardware IDs
		cmds := []string{
			"cat /proc/cpuinfo | grep 'Serial'",
			"cat /sys/class/dmi/id/product_uuid",
			"cat /sys/class/dmi/id/board_serial",
		}

		for _, cmd := range cmds {
			out, err := exec.Command("sh", "-c", cmd).Output()
			if err == nil {
				value := strings.TrimSpace(string(out))
				if value != "" {
					identifiers = append(identifiers, value)
				}
			}
		}
	}

	if len(identifiers) == 0 {
		return 0, fmt.Errorf("no hardware identifiers found")
	}

	// Combine all identifiers and create a hash
	combined := strings.Join(identifiers, "|")
	hash := sha256.Sum256([]byte(combined))
	// Return first 64 bits as an uint64 ==> this is the pseudo-unique identifier of the server
	return uint64(hash[7])<<56 + uint64(hash[6])<<48 + uint64(hash[5])<<40 + uint64(hash[4])<<32 + uint64(hash[3])<<24 + uint64(hash[2])<<16 + uint64(hash[1])<<8 + uint64(hash[0]), nil
}
