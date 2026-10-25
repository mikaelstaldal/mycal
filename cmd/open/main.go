// mycal-open is the desktop bridge for opening local iCalendar files.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mikaelstaldal/mycal/internal/clihttp"
)

const maxFileSize = 10 << 20

const bridgeStyle = "body{font:16px system-ui,sans-serif;max-width:38rem;margin:4rem auto;padding:1rem}button{font:inherit;padding:.5rem 1rem}"

var bridgePage = template.Must(template.New("bridge").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Open in MyCal</title><style>` + bridgeStyle + `</style></head><body>
<h1>Open iCalendar File</h1><p>Continue to MyCal to review and import this file.</p>
<button type="button">Continue to MyCal</button>
<script>
const target = {{.Target}};
const content = {{.Data}};
const nonce = {{.Nonce}};
const origin = new URL(target).origin;
let receiver;
document.querySelector('button').addEventListener('click', () => {
    receiver = window.open(target, '_blank');
    if (!receiver) document.querySelector('p').textContent = 'Allow popups for this page, then try again.';
});
window.addEventListener('message', event => {
    if (event.origin !== origin || event.source !== receiver ||
        event.data?.type !== 'mycal-desktop-ready' || event.data.nonce !== nonce) return;
    receiver.postMessage({type: 'mycal-desktop-file', nonce, content}, origin);
    document.querySelector('p').textContent = 'File sent to MyCal. You can close this tab.';
});
</script></body></html>`))

type bridgeData struct{ Target, Data, Nonce string }

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
	if len(data) == 0 || len(data) > maxFileSize {
		return errors.New("the iCalendar file must contain 1 byte to 10 MiB")
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) || !strings.HasPrefix(strings.TrimSpace(string(data)), "BEGIN:VCALENDAR") {
		return errors.New("the file must be a UTF-8 iCalendar document beginning with BEGIN:VCALENDAR")
	}
	return openBrowser(target, string(data))
}

func openBrowser(target, data string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()
	secret := rand.Text()
	address := listener.Addr().String()
	target += "#desktop-import=" + url.QueryEscape("http://"+address+"/"+secret)
	var page bytes.Buffer
	if err := bridgePage.Execute(&page, bridgeData{target, data, secret}); err != nil {
		return err
	}
	html := page.String()
	script := html[strings.Index(html, "<script>")+len("<script>") : strings.Index(html, "</script>")]
	scriptHash := sha256.Sum256([]byte(script))
	styleHash := sha256.Sum256([]byte(bridgeStyle))
	served := make(chan struct{}, 1)
	server := &http.Server{
		ReadTimeout: 10 * time.Second, ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler: http.MaxBytesHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != address || r.URL.Path != "/"+secret || r.Method != http.MethodGet {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'sha256-"+base64.StdEncoding.EncodeToString(scriptHash[:])+"'; style-src 'sha256-"+base64.StdEncoding.EncodeToString(styleHash[:])+"'; frame-ancestors 'none'; base-uri 'none'")
			if _, err := io.WriteString(w, html); err != nil {
				return
			}
			select {
			case served <- struct{}{}:
			default:
			}
		}), 1024),
	}
	defer func() { _ = server.Close() }()
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- server.Serve(listener) }()
	opener := browserCommand("http://" + address + "/" + secret)
	if err := opener.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	openerDone := make(chan error, 1)
	go func() { openerDone <- opener.Wait() }()
	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()
	for {
		select {
		case <-served:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return server.Shutdown(ctx)
		case err := <-openerDone:
			if err != nil {
				return fmt.Errorf("open browser: %w", err)
			}
			openerDone = nil
		case err := <-errorsCh:
			return err
		case <-timeout.C:
			return errors.New("timed out waiting for the browser")
		}
	}
}
