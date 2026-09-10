//go:build darwin

package terminal

import "syscall"

const (
	tcgets  = syscall.TIOCGETA
	tcsets  = syscall.TIOCSETA
	tcsetsw = syscall.TIOCSETAW
	tcsetsf = syscall.TIOCSETAF
)
