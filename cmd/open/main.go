// mycal-open is the desktop bridge for opening local iCalendar files.
package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/mikaelstaldal/mycal/internal/clihttp"
)

// Keep browser launch arguments bounded. Windows uses a lower shell URL limit.
const maxFileSize = 12 << 10

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mycal-open:", err)
		os.Exit(1)
	}
}

func targetURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("-url must be a MyCal HTTP(S) base URL without credentials, query or fragment")
	}
	loopback := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && (u.Scheme != "http" || !loopback) {
		return "", errors.New("use HTTPS for remote MyCal servers (HTTP is allowed for loopback only)")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/"
	u.RawPath = ""
	return u.String(), nil
}

func run(args []string) error {
	flags := flag.NewFlagSet("mycal-open", flag.ContinueOnError)
	base := flags.String("url", clihttp.URLDefault(), "MyCal base URL (default MYCAL_URL or http://127.0.0.1:8080)")
	install := flags.Bool("install", false, "register MyCal as an iCalendar application")
	setDefault := flags.Bool("set-default", false, "with -install, make MyCal the default iCalendar application")
	if err := flags.Parse(args); err != nil {
		return err
	}
	target, err := targetURL(*base)
	if err != nil {
		return err
	}
	if *install {
		if flags.NArg() != 0 {
			return errors.New("-install does not accept files")
		}
		return installDesktop(target, *setDefault)
	}
	if *setDefault || flags.NArg() != 1 {
		return errors.New("usage: mycal-open [-url URL] FILE.ics; or mycal-open [-url URL] -install [-set-default]")
	}
	f, err := os.Open(flags.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("the iCalendar file must be a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("the iCalendar file is empty")
	}
	if len(data) > maxFileSize {
		return importSizeError()
	}
	address, err := importURL(target, data)
	if err != nil {
		return err
	}
	if err := browserCommand(address).Run(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func importURL(target string, data []byte) (string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) || !strings.HasPrefix(strings.TrimSpace(string(data)), "BEGIN:VCALENDAR") {
		return "", errors.New("the file must be a UTF-8 iCalendar document beginning with BEGIN:VCALENDAR")
	}
	address := target + "#import=" + base64.RawURLEncoding.EncodeToString(data)
	if len(data) > maxFileSize || len(address) > maxBrowserURLLength {
		return "", importSizeError()
	}
	return address, nil
}

func importSizeError() error {
	return fmt.Errorf("event is too large for desktop URL import (maximum URL size %d bytes); use MyCal's Import Event file picker", maxBrowserURLLength)
}
