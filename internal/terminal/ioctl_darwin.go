//go:build darwin

package terminal

import "syscall"

const (
	tcgets     = syscall.TIOCGETA
	tcsetsf    = syscall.TIOCSETAF
	tiocgwinsz = syscall.TIOCGWINSZ
)

func setRawTermios(raw *syscall.Termios) {
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
}
