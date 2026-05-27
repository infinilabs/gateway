//go:build !(linux || darwin || freebsd || openbsd)

package floating_ip

import (
	"fmt"
	"net"
	"runtime"
)

func announceFloatingIP(_ net.IP, _ string) error {
	return fmt.Errorf("gratuitous arp announcement is not supported on %s", runtime.GOOS)
}
