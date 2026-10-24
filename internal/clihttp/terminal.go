package clihttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// CopyResponse protects terminal displays without changing exported files or pipes.
func CopyResponse(dst io.Writer, src io.Reader, terminal bool) (err error) {
	if !terminal {
		_, err := io.Copy(dst, src)
		return err
	}
	buffered := bufio.NewWriter(dst)
	defer func() { err = errors.Join(err, buffered.Flush()) }()
	reader := bufio.NewReader(src)
	for {
		r, size, err := reader.ReadRune()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var text string
		if r == unicode.ReplacementChar && size == 1 {
			// Invalid UTF-8 may include a terminal's single-byte C1 controls.
			if err := reader.UnreadRune(); err != nil {
				return err
			}
			b, err := reader.ReadByte()
			if err != nil {
				return err
			}
			text = fmt.Sprintf("\\x%02x", b)
		} else if unicode.IsControl(r) && r != '\t' && r != '\n' {
			quoted := strconv.QuoteRune(r)
			text = quoted[1 : len(quoted)-1]
		} else {
			text = string(r)
		}
		if _, err := io.WriteString(buffered, text); err != nil {
			return err
		}
	}
}

// EscapeTerminal escapes untrusted text for CLI diagnostics, including redirected stderr.
func EscapeTerminal(text string) string {
	var out strings.Builder
	_ = CopyResponse(&out, strings.NewReader(text), true)
	return out.String()
}
