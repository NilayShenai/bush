//go:build linux

package terminal

const (
	tcgets  = 0x5401
	tcsets  = 0x5402
	tcsetsw = 0x5403
	tcsetsf = 0x5404
)
