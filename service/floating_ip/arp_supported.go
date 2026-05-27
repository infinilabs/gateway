//go:build linux || darwin || freebsd || openbsd

package floating_ip

import (
	"net"

	"github.com/j-keck/arping"
)

func announceFloatingIP(ip net.IP, iface string) error {
	return arping.GratuitousArpOverIfaceByName(ip, iface)
}
