package terminal

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)



var (
	origTermios syscall.Termios
	isRaw       bool
	mu          sync.Mutex
)

type Winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func IsTerminal(fd int) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(tcgets),
		uintptr(unsafe.Pointer(&termios)),
		0, 0, 0,
	)
	return err == 0
}

func MakeRaw(fd int) (*syscall.Termios, error) {
	mu.Lock()
	defer mu.Unlock()

	var termios syscall.Termios
	_, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(tcgets),
		uintptr(unsafe.Pointer(&termios)),
		0, 0, 0,
	)
	if err != 0 {
		return nil, err
	}

	origTermios = termios

	raw := termios

	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON

	raw.Oflag &^= syscall.OPOST

	raw.Cflag |= syscall.CS8

	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG

	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	_, _, err = syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(tcsetsf),
		uintptr(unsafe.Pointer(&raw)),
		0, 0, 0,
	)
	if err != 0 {
		return nil, err
	}

	isRaw = true
	return &origTermios, nil
}

func Restore(fd int) error {
	mu.Lock()
	defer mu.Unlock()

	if !isRaw {
		return nil
	}

	_, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(tcsetsf),
		uintptr(unsafe.Pointer(&origTermios)),
		0, 0, 0,
	)
	if err != 0 {
		return err
	}

	isRaw = false
	return nil
}

func GetSize(fd int) (int, int, error) {
	var ws Winsize
	_, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
		0, 0, 0,
	)
	if err != 0 {
		return 80, 24, err
	}
	cols := int(ws.Col)
	rows := int(ws.Row)
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	return cols, rows, nil
}

func StdinFd() int {
	return int(os.Stdin.Fd())
}
