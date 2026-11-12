//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package kittygfx

import (
	"bytes"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// queryTerminal asks the terminal whether it understands the kitty graphics
// protocol and how large a cell is in pixels. It must run before tcell takes
// over the terminal, because it reads the replies from the tty itself.
//
// The query ends with a primary device attributes request (CSI c), which every
// terminal answers, so we know when to stop reading without waiting for the
// full timeout on terminals that ignore the graphics query.
func queryTerminal(timeout time.Duration) (supported bool, cellW, cellH int) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false, 0, 0
	}
	defer func() { _ = tty.Close() }()

	fd := int(tty.Fd())
	old, err := unix.IoctlGetTermios(fd, ioctlGetTermios)
	if err != nil {
		return false, 0, 0
	}

	raw := *old
	raw.Lflag &^= unix.ICANON | unix.ECHO
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 1 // reads return after at most 100ms
	if err := unix.IoctlSetTermios(fd, ioctlSetTermios, &raw); err != nil {
		return false, 0, 0
	}
	defer func() { _ = unix.IoctlSetTermios(fd, ioctlSetTermios, old) }()

	query := "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\" + "\x1b[16t" + "\x1b[c"
	if _, err := tty.WriteString(query); err != nil {
		return false, 0, 0
	}

	var reply []byte
	buf := make([]byte, 256)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		n, _ := tty.Read(buf)
		reply = append(reply, buf[:n]...)
		if hasDeviceAttributes(reply) {
			break
		}
	}

	return parseReply(reply)
}

// hasDeviceAttributes reports whether the reply contains a DA1 response
// (ESC [ ? ... c), which marks the end of our query.
func hasDeviceAttributes(b []byte) bool {
	i := bytes.Index(b, []byte("\x1b[?"))
	if i < 0 {
		return false
	}
	return bytes.IndexByte(b[i:], 'c') >= 0
}
