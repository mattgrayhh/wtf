//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package kittygfx

import "time"

// queryTerminal is not implemented on this platform; images fall back to text blocks.
func queryTerminal(_ time.Duration) (bool, int, int) {
	return false, 0, 0
}
