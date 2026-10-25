package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

type termState struct {
	old    syscall.Termios
	fd     uintptr
	active bool
}

func rawTerminal() (*termState, error) {
	fd := os.Stdin.Fd()
	t := &termState{fd: fd}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&t.old))); e != 0 {
		return nil, e
	}
	n := t.old
	n.Lflag &^= syscall.ICANON | syscall.ECHO
	n.Cc[syscall.VMIN] = 1
	n.Cc[syscall.VTIME] = 0
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&n))); e != 0 {
		return nil, e
	}
	t.active = true
	return t, nil
}
func (t *termState) restore() {
	if t != nil && t.active {
		syscall.Syscall(syscall.SYS_IOCTL, t.fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&t.old)))
		t.active = false
	}
}

type voiceCapture struct {
	cmd     *exec.Cmd
	path    string
	started time.Time
	once    sync.Once
}

func startCapture() (*voiceCapture, error) {
	f, err := os.CreateTemp("", "keimling-*.pcm")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	_ = f.Close()
	// Raw PCM avoids an incomplete WAV header when arecord is interrupted.
	cmd := exec.Command("arecord", "-q", "-f", "S16_LE", "-r", "12000", "-c", "1", "-d", "900", "-t", "raw", path)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("start arecord: %w (install alsa-utils and check the capture device)", err)
	}
	return &voiceCapture{cmd: cmd, path: path, started: time.Now()}, nil
}
func (c *voiceCapture) finish() (string, error) {
	if c == nil {
		return "", fmt.Errorf("no recording")
	}
	if c.cmd.ProcessState == nil {
		_ = c.cmd.Process.Signal(os.Interrupt)
	}
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = c.cmd.Process.Kill()
		<-done
	}
	wav, err := pcmFileToWAV(c.path, 12000)
	_ = os.Remove(c.path)
	return wav, err
}

// Read one editable line. F1 starts/stops recording; Space stops while recording.
// /voice starts recording after Enter. Transcription remains an editable draft.
func readInteractiveLine(reader *bufio.Reader, client *Client, model string) (string, error) {
	state, e := rawTerminal()
	if e != nil {
		fmt.Print("[36m❯[0m ")
		s, e := reader.ReadString('\n')
		return strings.TrimSpace(s), e
	}
	fmt.Print("\x1b[?2004h")
	defer func() { fmt.Print("\x1b[?2004l"); state.restore() }()
	draft := []rune{}
	cursor := 0
	history := lineHistory
	historyPos := len(history)
	savedDraft := ""
	var recording *voiceCapture
	// Keep track of the rendered cursor row: a wrapped prompt occupies several
	// terminal rows and clearing only the current row leaves stale copies.
	previousCursorRow := 0
	repaint := func() {
		if recording != nil {
			return
		}
		width := terminalColumns(fdForTerminal())
		text, row, col := layoutPrompt(draft, cursor, width)
		_, endRow, _ := layoutPrompt(draft, len(draft), width)
		fmt.Print("\r")
		if previousCursorRow > 0 {
			fmt.Printf("\033[%dA", previousCursorRow)
		}
		fmt.Print("\033[J\033[36m❯\033[0m ", text)
		if endRow > row {
			fmt.Printf("\033[%dA", endRow-row)
		}
		fmt.Print("\r")
		if col > 0 {
			fmt.Printf("\033[%dC", col)
		}
		previousCursorRow = row
	}
	repaint()
	transcribe := func(send bool) (string, bool) {
		p, e := recording.finish()
		recording = nil
		if e != nil {
			fmt.Printf("\nRecording: %v\n", e)
			repaint()
			return "", false
		}
		defer os.Remove(p)
		fmt.Print("\r\033[2K\033[36m◌\033[0m Transcribing ...")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		txt, e := "", fmt.Errorf("OpenAI transcription unavailable: external services disabled or credentials missing")
		if client != nil && client.APIKey != "" {
			txt, e = client.Transcribe(ctx, p, model)
		}
		cancel()
		if e != nil {
			fmt.Printf("Transcription: %v\n", e)
		} else if txt != "" {
			if len(draft) > 0 && strings.TrimSpace(string(draft)) != strings.TrimSpace(txt) {
				draft = insertRune(draft, cursor, ' ')
				cursor++
			}
			if strings.TrimSpace(string(draft)) != strings.TrimSpace(txt) {
				draft = append(draft, []rune(txt)...)
				cursor = len(draft)
			}
		}
		fmt.Print("\r\033[2K")
		if send && e == nil && strings.TrimSpace(string(draft)) != "" {
			text := strings.TrimSpace(string(draft))
			lineHistory = append(lineHistory, text)
			if len(lineHistory) > 100 {
				lineHistory = lineHistory[len(lineHistory)-100:]
			}
			fmt.Printf("\033[36m❯\033[0m %s\n", text)
			return text, true
		}
		repaint()
		return "", false
	}
	// Exactly one synchronous stdin reader. Never leave a blocked reader behind.
	// A per-prompt goroutine previously stole keys from later approval prompts.
	fd := int(os.Stdin.Fd())
	nextKey := func() (byte, error) { return readTerminalByte(fd, 0) }
	for {
		var timeout time.Duration
		if recording != nil {
			timeout = time.Until(recording.started.Add(15 * time.Minute))
			if timeout <= 0 {
				transcribe(false)
				continue
			}
		}
		if activeControl != nil {
			if activeControl.state.Load() == 3 {
				fmt.Print("\x1b[?2004l")
				state.restore()
				return "", errApprovalPending
			}
			if activeControl.state.Load() == 1 {
				// Restore the real terminal before signalling that exclusive input is free.
				fmt.Print("\x1b[?2004l")
				state.restore()
				activeControl.state.Store(2)
				activeControl.resume <- struct{}{}
				return "", errUIHandoff
			}
			select {
			case <-activeControl.finished:
				return "", errAgentIdle
			default:
			}
			if timeout == 0 || timeout > 100*time.Millisecond {
				timeout = 100 * time.Millisecond
			}
		}
		b, e := readTerminalByte(fd, timeout)
		if e == errKeyTimeout {
			if activeControl != nil {
				if recording != nil && time.Since(recording.started) >= 15*time.Minute {
					transcribe(false)
				}
				continue
			}
			transcribe(false)
			continue
		}
		if e != nil {
			if recording != nil {
				p, _ := recording.finish()
				os.Remove(p)
			}
			fmt.Println()
			return "", e
		}
		if b == 27 { // F1 commonly ESC O P or ESC [ 11 ~
			next, e := nextKey()
			if e != nil {
				return "", e
			}
			f1 := false
			seq := ""
			if next == 'O' {
				last, _ := nextKey()
				f1 = last == 'P'
			} else if next == '[' {
				for len(seq) < 8 {
					q, er := nextKey()
					if er != nil {
						break
					}
					seq += string(q)
					if (q >= 'A' && q <= 'Z') || q == '~' {
						break
					}
				}
				f1 = seq == "11~" || seq == "[11~"
			}
			if !f1 && recording == nil && next == '[' && seq == "200~" {
				// Bracketed paste: internal newlines are content, not submission.
				data, er := readBracketedPaste(fd)
				if er != nil {
					return "", er
				}
				for _, r := range data {
					draft = insertRune(draft, cursor, r)
					cursor++
				}
				repaint()
				continue
			}
			if !f1 && recording == nil && next == '[' {
				switch seq {
				case "D":
					if cursor > 0 {
						cursor--
					}
				case "C":
					if cursor < len(draft) {
						cursor++
					}
				case "H", "1~", "7~":
					cursor = 0
				case "F", "4~", "8~":
					cursor = len(draft)
				case "3~":
					if cursor < len(draft) {
						draft = append(draft[:cursor], draft[cursor+1:]...)
					}
				case "A":
					if historyPos == len(history) {
						savedDraft = string(draft)
					}
					if historyPos > 0 {
						historyPos--
						draft = []rune(history[historyPos])
						cursor = len(draft)
					}
				case "B":
					if historyPos < len(history) {
						historyPos++
						if historyPos == len(history) {
							draft = []rune(savedDraft)
						} else {
							draft = []rune(history[historyPos])
						}
						cursor = len(draft)
					}
				}
			}
			if f1 {
				if recording != nil {
					transcribe(false)
				} else {
					recording, e = startCapture()
					if e != nil {
						fmt.Printf("\nMicrophone: %v\n", e)
					} else {
						fmt.Print("\r\033[2K\033[31m● Recording\033[0m · F1/Space to stop · max 15 min")
					}
					repaint()
				}
			}
			repaint()
			continue
		}
		if recording != nil {
			if b == ' ' || b == '\r' || b == '\n' {
				if text, ok := transcribe(submitRecordingKey(b)); ok {
					return text, nil
				}
			} else if b == 3 {
				p, _ := recording.finish()
				os.Remove(p)
				recording = nil
				fmt.Print("\nRecording discarded\n")
				repaint()
			}
			continue
		}
		if b == 7 { // Ctrl+G toggles autopilot (Ctrl+A is already line-home).
			fmt.Println("\r\033[2K")
			return "/autopilot toggle", nil
		}
		switch b {
		case '\t':
			next, pos, matches := completeSlash(string(draft), cursor)
			if len(matches) > 0 {
				draft, cursor = []rune(next), pos
				if len(matches) > 1 {
					// Display available choices when completion is ambiguous.
					fmt.Print("\n" + strings.Join(matches, "  ") + "\n")
					previousCursorRow = 0
				}
			}
		case '\r', '\n':
			if len(draft) == 0 {
				recording, e = startCapture()
				if e != nil {
					fmt.Printf("\nMicrophone: %v\n", e)
					repaint()
				} else {
					fmt.Print("\r\033[2K\033[31m● Recording\033[0m · F1/Space/Enter to stop · max 15 min")
				}
				continue
			}
			text := strings.TrimSpace(string(draft))
			fmt.Print("\n")
			if text == "/voice" {
				recording, e = startCapture()
				if e != nil {
					fmt.Printf("Microphone: %v\n", e)
				} else {
					fmt.Print("\r\033[2K\033[31m● Recording\033[0m · F1/Space to stop · max 15 min")
				}
				draft = nil
				cursor = 0
				repaint()
				continue
			}
			if text != "" {
				lineHistory = append(lineHistory, text)
				if len(lineHistory) > 100 {
					lineHistory = lineHistory[len(lineHistory)-100:]
				}
			}
			return text, nil
		case ' ':
			if len(draft) == 0 {
				recording, e = startCapture()
				if e != nil {
					fmt.Printf("\nMicrophone: %v\n", e)
					repaint()
				} else {
					fmt.Print("\r\033[2K\033[31m● Recording\033[0m · F1/Space to stop · max 15 min")
				}
				continue
			}
			draft = insertRune(draft, cursor, ' ')
			cursor++
		case 127, 8:
			if cursor > 0 {
				draft = append(draft[:cursor-1], draft[cursor:]...)
				cursor--
			}
		case 1:
			cursor = 0
		case 5:
			cursor = len(draft)
		case 3:
			if activeControl != nil {
				activeControl.cancel()
				fmt.Print("\nTask interrupted\n")
				return "", errAgentIdle
			}
			fmt.Print("\n")
			return "", nil
		case 4:
			if len(draft) == 0 {
				fmt.Print("\n")
				return "", io.EOF
			}
		default:
			if b >= 32 && b < 128 {
				draft = insertRune(draft, cursor, rune(b))
				cursor++
			} else if b >= 128 {
				buf := []byte{b}
				want := 0
				switch {
				case b&0xe0 == 0xc0:
					want = 2
				case b&0xf0 == 0xe0:
					want = 3
				case b&0xf8 == 0xf0:
					want = 4
				}
				for len(buf) < want {
					c, err := nextKey()
					if err != nil {
						break
					}
					buf = append(buf, c)
				}
				if utf8.Valid(buf) {
					draft = insertRune(draft, cursor, []rune(string(buf))[0])
					cursor++
				}
			}
		}
		repaint()
	}
}

var errKeyTimeout = fmt.Errorf("keyboard timeout")
var activeControl *taskControl

func readTerminalByte(fd int, timeout time.Duration) (byte, error) {
	if timeout > 0 {
		var set syscall.FdSet
		if fd < 0 || fd >= len(set.Bits)*64 {
			return 0, fmt.Errorf("terminal fd out of range")
		}
		set.Bits[fd/64] |= 1 << uint(fd%64)
		tv := syscall.NsecToTimeval(timeout.Nanoseconds())
		n, err := syscall.Select(fd+1, &set, nil, nil, &tv)
		if err == syscall.EINTR {
			return 0, errKeyTimeout
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, errKeyTimeout
		}
	}
	var one [1]byte
	for {
		n, e := syscall.Read(fd, one[:])
		if e == syscall.EINTR {
			continue
		}
		if e != nil {
			return 0, e
		}
		if n == 0 {
			return 0, io.EOF
		}
		return one[0], nil
	}
}

func discardPendingInput() {
	// TCIFLUSH: discard keystrokes accumulated while the agent was busy.
	syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(0x540B), 0)
}

func insertRune(r []rune, pos int, ch rune) []rune {
	r = append(r, 0)
	copy(r[pos+1:], r[pos:])
	r[pos] = ch
	return r
}

var lineHistory []string

// Enter submits, while Space and F1 keep the transcription as an editable draft.
func submitRecordingKey(key byte) bool { return key == '\r' || key == '\n' }

// readBracketedPaste consumes bytes until the terminal's exact paste terminator.
// No line is submitted while collecting pasted content.
func readBracketedPaste(fd int) (string, error) {
	const end = "\x1b[201~"
	var data []byte
	for len(data) < 1<<20 {
		b, err := readTerminalByte(fd, 0)
		if err != nil {
			return "", err
		}
		data = append(data, b)
		if len(data) >= len(end) && string(data[len(data)-len(end):]) == end {
			return strings.ReplaceAll(string(data[:len(data)-len(end)]), "\r\n", "\n"), nil
		}
	}
	return "", fmt.Errorf("paste exceeds 1 MiB")
}

// readApproval handles keyboard input on the UI goroutine only.
// It intentionally does not use the regular chat editor (where Enter starts audio).
func readApproval(ctx context.Context) bool {
	discardPendingInput()
	state, err := rawTerminal()
	if err != nil {
		fmt.Println("Approval terminal unavailable:", err)
		return false
	}
	defer state.restore()
	fmt.Print("\r\033[2KApprove? [Y/n] ")
	var input []byte
	for {
		if ctx.Err() != nil {
			fmt.Println()
			return false
		}
		b, err := readTerminalByte(int(os.Stdin.Fd()), 100*time.Millisecond)
		if err == errKeyTimeout {
			continue
		}
		if err != nil {
			fmt.Println()
			return false
		}
		switch b {
		case 3:
			fmt.Println("^C")
			return false
		case 13, 10:
			fmt.Println()
			answer := strings.ToLower(strings.TrimSpace(string(input)))
			return answer == "" || answer == "y" || answer == "yes" || answer == "j" || answer == "ja"
		case 127, 8:
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
		default:
			if b >= 32 && b < 127 && len(input) < 16 {
				input = append(input, b)
				fmt.Printf("%c", b)
			}
		}
	}
}

// fdForTerminal obtains the active terminal without caching terminal width.
func fdForTerminal() int { return int(os.Stdin.Fd()) }

func terminalColumns(fd int) int {
	var ws struct{ Rows, Cols, Xpixel, Ypixel uint16 }
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno != 0 || ws.Cols < 12 {
		return 80
	}
	return int(ws.Cols)
}

// runeColumns is sufficient for common Unicode terminals, including CJK and emoji.
func runeColumns(r rune) int {
	if r == '\n' {
		return 0
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || r == 0x200d || (r >= 0xfe00 && r <= 0xfe0f) {
		return 0
	}
	if r >= 0x1100 && (r <= 0x115f || r >= 0x2329 && r <= 0x232a || r >= 0x2e80 && r <= 0xa4cf || r >= 0xac00 && r <= 0xd7a3 || r >= 0xf900 && r <= 0xfaff || r >= 0xfe10 && r <= 0xfe6f || r >= 0xff00 && r <= 0xff60 || r >= 0xffe0 && r <= 0xffe6 || r >= 0x1f300 && r <= 0x1faff) {
		return 2
	}
	return 1
}

// layoutPrompt explicitly wraps at width-1, never relying on the terminal's
// delayed autowrap state. Return cursor coordinates within the rendered prompt.
func layoutPrompt(draft []rune, cursor, width int) (string, int, int) {
	if width < 12 {
		width = 80
	}
	limit := width - 1
	row, col := 0, 2 // visible width of "❯ "
	cursorRow, cursorCol := 0, 2
	var out strings.Builder
	for i, r := range draft {
		if i == cursor {
			cursorRow, cursorCol = row, col
		}
		if r == '\n' {
			out.WriteString("\r\n")
			row++
			col = 0
			continue
		}
		w := runeColumns(r)
		if col+w > limit {
			out.WriteString("\r\n")
			row++
			col = 0
		}
		out.WriteRune(r)
		col += w
	}
	if cursor >= len(draft) {
		cursorRow, cursorCol = row, col
	}
	return out.String(), cursorRow, cursorCol
}
