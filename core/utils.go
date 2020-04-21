package core

import (
	"net"

	"github.com/asaskevich/govalidator"
)

// IsValidIPAddressOrHostname returns whether the provided input is a valid IP address or hostname
func IsValidIPAddressOrHostname(address string) bool {
	addr := net.ParseIP(address)
	if addr != nil {
		return true
	}

	return govalidator.IsDNSName(address)
}
