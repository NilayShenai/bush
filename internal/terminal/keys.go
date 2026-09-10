package terminal

import (
	"io"
)

type KeyType int

const (
	KeyRune KeyType = iota
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyTab
	KeyShiftTab
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlK
	KeyCtrlL
	KeyCtrlN
	KeyCtrlP
	KeyCtrlR
	KeyCtrlU
	KeyCtrlW
	KeyCtrlZ
	KeyAltLeft
	KeyAltRight
	KeyEscape
	KeyUnknown
)

type KeyEvent struct {
	Type KeyType
	Rune rune
}

func ReadKey(r io.Reader) (KeyEvent, error) {
	var buf [1]byte
	n, err := r.Read(buf[:])
	if err != nil {
		return KeyEvent{Type: KeyUnknown}, err
	}
	if n == 0 {
		return KeyEvent{Type: KeyUnknown}, nil
	}

	b := buf[0]

	switch b {
	case 1:
		return KeyEvent{Type: KeyCtrlA}, nil
	case 2:
		return KeyEvent{Type: KeyCtrlB}, nil
	case 3:
		return KeyEvent{Type: KeyCtrlC}, nil
	case 4:
		return KeyEvent{Type: KeyCtrlD}, nil
	case 5:
		return KeyEvent{Type: KeyCtrlE}, nil
	case 6:
		return KeyEvent{Type: KeyCtrlF}, nil
	case 9:
		return KeyEvent{Type: KeyTab}, nil
	case 11:
		return KeyEvent{Type: KeyCtrlK}, nil
	case 12:
		return KeyEvent{Type: KeyCtrlL}, nil
	case 13, 10:
		return KeyEvent{Type: KeyEnter}, nil
	case 14:
		return KeyEvent{Type: KeyCtrlN}, nil
	case 16:
		return KeyEvent{Type: KeyCtrlP}, nil
	case 18:
		return KeyEvent{Type: KeyCtrlR}, nil
	case 21:
		return KeyEvent{Type: KeyCtrlU}, nil
	case 23:
		return KeyEvent{Type: KeyCtrlW}, nil
	case 26:
		return KeyEvent{Type: KeyCtrlZ}, nil
	case 127, 8:
		return KeyEvent{Type: KeyBackspace}, nil
	}

	if b == 27 {
		var next [1]byte
		n, err := r.Read(next[:])
		if err != nil || n == 0 {
			return KeyEvent{Type: KeyEscape}, nil
		}

		if next[0] == 'b' {
			return KeyEvent{Type: KeyAltLeft}, nil
		}
		if next[0] == 'f' {
			return KeyEvent{Type: KeyAltRight}, nil
		}

		if next[0] == 'O' {

			var ss3 [1]byte
			n, err := r.Read(ss3[:])
			if err == nil && n > 0 {
				switch ss3[0] {
				case 'H':
					return KeyEvent{Type: KeyHome}, nil
				case 'F':
					return KeyEvent{Type: KeyEnd}, nil
				}
			}
			return KeyEvent{Type: KeyEscape}, nil
		}

		if next[0] == '[' {

			var csi []byte
			for {
				var ch [1]byte
				n, err := r.Read(ch[:])
				if err != nil || n == 0 {
					break
				}
				csi = append(csi, ch[0])

				if ch[0] >= 0x40 && ch[0] <= 0x7E {
					break
				}
			}

			if len(csi) == 0 {
				return KeyEvent{Type: KeyEscape}, nil
			}

			last := csi[len(csi)-1]

			if len(csi) == 1 {
				switch last {
				case 'A':
					return KeyEvent{Type: KeyArrowUp}, nil
				case 'B':
					return KeyEvent{Type: KeyArrowDown}, nil
				case 'C':
					return KeyEvent{Type: KeyArrowRight}, nil
				case 'D':
					return KeyEvent{Type: KeyArrowLeft}, nil
				case 'H':
					return KeyEvent{Type: KeyHome}, nil
				case 'F':
					return KeyEvent{Type: KeyEnd}, nil
				case 'Z':
					return KeyEvent{Type: KeyShiftTab}, nil
				}
			}

			if last == 'C' {
				return KeyEvent{Type: KeyAltRight}, nil
			}
			if last == 'D' {
				return KeyEvent{Type: KeyAltLeft}, nil
			}

			if last == '~' {
				first := csi[0]
				switch first {
				case '1', '7':
					return KeyEvent{Type: KeyHome}, nil
				case '3':
					return KeyEvent{Type: KeyDelete}, nil
				case '4', '8':
					return KeyEvent{Type: KeyEnd}, nil
				case '5':
					return KeyEvent{Type: KeyPageUp}, nil
				case '6':
					return KeyEvent{Type: KeyPageDown}, nil
				}
			}

			return KeyEvent{Type: KeyUnknown}, nil
		}

		return KeyEvent{Type: KeyEscape}, nil
	}

	return KeyEvent{Type: KeyRune, Rune: rune(b)}, nil
}
