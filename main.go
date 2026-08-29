package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/mikaelstaldal/go-server-common/auth"
	"github.com/mikaelstaldal/go-server-common/csrf"
	"github.com/mikaelstaldal/go-server-common/httputil"
	commonsqlite "github.com/mikaelstaldal/go-server-common/sqlite"
	commonweb "github.com/mikaelstaldal/go-server-common/web"
	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/web"
)

const databaseName = "mycal.sqlite"

// deriveSiblingURL returns the base URL of a sibling app served from the same
// origin, derived from publicURL by replacing its path with siblingPath.
// Returns empty string if publicURL is empty or has no path segment — a MyCal
// deployed at the origin root leaves no room for siblings, so nothing is
// assumed about them.
func deriveSiblingURL(publicURL, siblingPath string) string {
	if publicURL == "" {
		return ""
	}
	u, err := url.Parse(publicURL)
	if err != nil || strings.Trim(u.Path, "/") == "" {
		return ""
	}
	u.Path = siblingPath
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// deriveMymailURL returns the MyMail base URL derived from publicURL.
func deriveMymailURL(publicURL string) string {
	return deriveSiblingURL(publicURL, "/mymail")
}

// deriveMynotesURL returns the MyNotes base URL derived from publicURL.
func deriveMynotesURL(publicURL string) string {
	return deriveSiblingURL(publicURL, "/mynotes")
}

// serverConfig is the deployment configuration handed to the web UI through an
// injected inline <script> (see web/ts/util/serverconfig.ts). Every field is
// omitted when unset, so a plain deployment with no siblings injects nothing at
// all.
type serverConfig struct {
	// MymailURL is the base URL of the sibling MyMail instance, or "" when the
	// integration is not configured.
	MymailURL string `json:"mymailUrl,omitempty"`
	// MynotesURL is the base URL of the sibling MyNotes instance, or "" when
	// the integration is not configured.
	MynotesURL string `json:"mynotesUrl,omitempty"`
	// Demo marks the backend-less demo build, where a service worker emulates
	// the REST API against browser-local storage (see web/ts/demo/).
	Demo bool `json:"demo,omitempty"`
}

// script returns an inline JS snippet that sets window.__serverConfig, or "" when
// there is nothing to configure. The snippet is spliced into index.html
// verbatim, so the values must not be able to terminate the surrounding
// <script> element. json.Marshal emits <, > and & as Unicode escapes (HTML
// escaping is on by default), which makes that impossible — do not replace it
// with an encoder that has SetEscapeHTML(false).
func (c serverConfig) script() string {
	if c == (serverConfig{}) {
		return ""
	}
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return "window.__serverConfig=" + string(b) + ";"
}

// inlineScriptCSPHash returns the CSP sha256 hash token for an inline script.
func inlineScriptCSPHash(script string) string {
	h := sha256.Sum256([]byte(script))
	return "'sha256-" + base64.StdEncoding.EncodeToString(h[:]) + "'"
}

// buildIndexHTML reads index.html from the embedded FS and applies the two
// deployment-time rewrites: a <base href> for a subpath deployment, and the
// injected inline <script> carrying the server configuration. It returns the
// rewritten page plus the CSP script-src addition that script needs (empty when
// nothing was injected).
//
// index.html carries no <base> of its own — every URL in it is relative, so a
// page served under any path resolves them correctly against its own location.
// The tag is therefore only added when basePath says otherwise, which today is
// the static demo bundle, whose index.html is also reachable as a directory
// index ("/mycal/" as well as "/mycal/index.html").
//
// Shared by the server and the demo paths so the two can never drift.
func buildIndexHTML(staticFS fs.FS, basePath string, cfg serverConfig) (html []byte, configScriptSrc string, err error) {
	content, err := fs.ReadFile(staticFS, "static/index.html")
	if err != nil {
		return nil, "", err
	}
	if basePath != "" && basePath != "/" {
		content = []byte(strings.Replace(string(content), "<head>",
			"<head>\n    <base href=\""+basePath+"\">", 1))
	}
	if script := cfg.script(); script != "" {
		configScriptSrc = inlineScriptCSPHash(script)
		content = []byte(strings.Replace(string(content), "</head>",
			"<script>"+script+"</script>\n</head>", 1))
	}
	return content, configScriptSrc, nil
}

// contentSecurityPolicy returns the Content-Security-Policy header value for the
// app, with scriptSrc holding the script-src source expressions beyond the fixed
// ones. Shared by the real server and the demo server.
func contentSecurityPolicy(scriptSrc string) string {
	// frame-ancestors is header-only: a browser ignores it in a <meta> element,
	// which is why it sits here rather than in the shared part below.
	return metaContentSecurityPolicy(scriptSrc) + "; frame-ancestors 'none'"
}

// metaContentSecurityPolicy returns the part of the policy that is equally valid
// in a <meta http-equiv="Content-Security-Policy"> element, which is how the
// static demo bundle carries it: a bundle is served by someone else's web
// server, so there is no response header of ours to put it in.
//
// The demo's service worker needs no directive of its own: worker-src falls
// back to default-src 'self', which already covers a same-origin script.
func metaContentSecurityPolicy(scriptSrc string) string {
	return "default-src 'self';" +
		" script-src " + scriptSrc + " https://maps.googleapis.com;" +
		" style-src-elem 'self'; style-src-attr 'unsafe-inline';" +
		" img-src 'self' data: https://*.tile.openstreetmap.org https://maps.googleapis.com https://maps.gstatic.com;" +
		" connect-src 'self' https://maps.googleapis.com https://*.tile.openstreetmap.org;" +
		" font-src 'self';" +
		// The MyNotes render kit (/mynotes/render/) is framed to display a
		// linked note; it is served from this same origin, so 'self' covers it.
		" frame-src 'self';" +
		" object-src 'none'"
}

// injectMetaCSP adds a Content-Security-Policy <meta> element to the page, right
// after the charset declaration so it precedes every script and style the page
// pulls in.
func injectMetaCSP(html []byte, csp string) []byte {
	const charset = `<meta charset="UTF-8">`
	meta := charset + "\n    " + `<meta http-equiv="Content-Security-Policy" content="` + csp + `">`
	return bytes.Replace(html, []byte(charset), []byte(meta), 1)
}

// basePathChars restricts the base path to unreserved URL characters plus the
// separator. The path is spliced into <base href>, so anything that could close
// the attribute or introduce another URL component is refused outright rather
// than escaped.
var basePathChars = regexp.MustCompile(`^[A-Za-z0-9._~/-]+$`)

// basePathFromPublicURL returns the path the app is served under, derived from
// publicURL and always ending in "/". A public URL with no path (or none at all)
// yields "/", the origin root. Mirrors the same helper in MyNotes.
func basePathFromPublicURL(publicURL string) (string, error) {
	if publicURL == "" {
		return "/", nil
	}
	u, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("parse public URL %q: %w", publicURL, err)
	}
	p := u.Path
	if p == "" || p == "/" {
		return "/", nil
	}
	if !basePathChars.MatchString(p) {
		return "", fmt.Errorf("public URL path %q must contain only the characters A-Z a-z 0-9 . _ ~ - /", p)
	}
	// A leading "//" would make <base href> protocol-relative, re-pointing the
	// whole UI at another host.
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return "", fmt.Errorf("public URL path %q must start with a single %q", p, "/")
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p, nil
}

// requireEmptyDir accepts a path that does not exist or is an empty directory,
// and rejects anything else.
func requireEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check %s: %w", dir, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s already exists and is not empty; pass a new directory", dir)
	}
	return nil
}

// writeDemoBundle assembles a self-contained static demo site in outDir and
// returns. The result is plain files: any web server that serves a directory can
// host it, and the service worker (demo-sw.js) stands in for the REST API,
// keeping every event, calendar and preference in browser-local storage.
//
// outDir must not already exist, or must be an empty directory — the bundle is
// never merged into a populated directory, so a stale file from an earlier
// version cannot linger and no pre-existing content is overwritten.
//
// publicURL is honoured as it is for the server, except that its path component
// additionally becomes the page's <base href>: a bundle destined for
// https://example.com/mycal/ is built with -public-url https://example.com/mycal.
// Every URL the frontend builds is resolved against that base (see
// web/ts/api/client.ts and web/ts/demo-client.ts), so the same files work at the
// origin root and under a path.
func writeDemoBundle(outDir, publicURL string) error {
	if err := requireEmptyDir(outDir); err != nil {
		return err
	}
	basePath, err := basePathFromPublicURL(publicURL)
	if err != nil {
		return err
	}

	importMapHash, err := commonweb.ImportMapCSPHash(web.Static)
	if err != nil {
		return fmt.Errorf("compute importmap CSP hash: %w", err)
	}
	indexHTML, configScriptHash, err := buildIndexHTML(web.Static, basePath, serverConfig{Demo: true})
	if err != nil {
		return fmt.Errorf("build index.html: %w", err)
	}
	// A static bundle has no server of ours to set response headers, so the
	// policy the demo server would send travels in the page instead.
	indexHTML = injectMetaCSP(indexHTML,
		metaContentSecurityPolicy("'self' "+importMapHash+" "+configScriptHash))

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outDir, err)
	}
	files := 0
	err = fs.WalkDir(web.Static, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(path, "static"), "/")
		dest := filepath.Join(outDir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := fs.ReadFile(web.Static, path)
		if err != nil {
			return err
		}
		if rel == "index.html" {
			data = indexHTML
		}
		files++
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	fmt.Printf("Wrote a static MyCal demo (%d files) to %s\n", files, outDir)
	fmt.Printf("Serve that directory with any web server; it needs no backend.\n")
	if basePath == "/" {
		fmt.Printf("It is built for the origin root — to deploy it under a path, rebuild with -public-url.\n")
	} else {
		fmt.Printf("It is built for the path %s (from -public-url).\n", basePath)
	}
	fmt.Printf("A service worker is required, so serve it over HTTPS or from localhost.\n")
	return nil
}

// flagWasSet reports whether the named flag was given on the command line, as
// opposed to holding its default value.
func flagWasSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// demoAPIUnavailable answers any REST API request that reaches the server in
// demo mode, which only happens when the browser's service worker is not
// installed or not in control.
func demoAPIUnavailable(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"demo mode: the in-browser backend is not running; reload the page"}`))
}

func main() {
	version := flag.Bool("version", false, "print version information and exit")
	port := flag.Int("port", 8080, "port to listen on")
	addr := flag.String("addr", "127.0.0.1", "address to listen on")
	dataDir := flag.String("data", "data", "directory to store data in")
	basicAuthFile := flag.String("basic-auth-file", "", "enable HTTP basic auth with username and password from given file in htpasswd format (bcrypt only)")
	basicAuthRealm := flag.String("basic-auth-realm", "mycal", "realm for HTTP basic auth")
	httpsMode := flag.Bool("https", false, "set Strict-Transport-Security header (use when served behind a TLS-terminating proxy)")
	publicURL := flag.String("public-url", "", "Public-facing base URL for CSRF validation, e.g. https://example.com (defaults to http://<addr>:<port>)")
	exportICS := flag.String("export-ics", "", "export all events to an .ics file and exit")
	demoServer := flag.Bool("demo-server", false, "serve the backend-less demo: no database is opened, the browser stores all data locally")
	demoBundle := flag.String("demo-bundle", "", "write a self-contained static demo site to this new directory and exit (takes no -data)")
	flag.Parse()

	if *version {
		printVersion()
		return
	}

	if *port < 1 || *port > 65535 {
		log.Fatalf("Invalid port number: %d. Must be between 1 and 65535", *port)
	}

	opts := httpServerOptions{
		addr:           *addr,
		port:           *port,
		publicURL:      *publicURL,
		basicAuthFile:  *basicAuthFile,
		basicAuthRealm: *basicAuthRealm,
		httpsMode:      *httpsMode,
	}

	// Demo mode is handled before anything touches storage: it has no database
	// at all, so a stray -data would be silently ignored rather than honoured.
	if *demoServer || *demoBundle != "" {
		if *demoServer && *demoBundle != "" {
			log.Fatalf("-demo-server and -demo-bundle do different things; pass one of them")
		}
		if flagWasSet("data") {
			log.Fatalf("demo mode stores nothing on the server; remove -data")
		}
		if *exportICS != "" {
			log.Fatalf("demo mode has no database to export; remove -export-ics")
		}
		if *demoBundle != "" {
			if err := writeDemoBundle(*demoBundle, *publicURL); err != nil {
				log.Fatalf("%v", err)
			}
			return
		}
		if err := runDemoServer(opts); err != nil {
			log.Fatalf("%v", err)
		}
		return
	}

	info, err := os.Stat(*dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(*dataDir, 0700); err != nil {
				log.Fatalf("Could not create data directory: %s", *dataDir)
			}
		} else {
			log.Fatalf("Failed to access data directory %s: %v", *dataDir, err)
		}
	} else {
		if !info.IsDir() {
			log.Fatalf("Data directory path is not a directory: %s", *dataDir)
		}
	}
	databaseFile := filepath.Join(*dataDir, databaseName)

	info, err = os.Stat(databaseFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("Failed to access database file %s: %v", databaseFile, err)
		}
	} else {
		if !info.Mode().IsRegular() {
			log.Fatalf("Database file is not a regular file: %s", databaseFile)
		}
	}

	if *exportICS != "" {
		// Open database read-only so this can run concurrently with a server
		db, err := sql.Open("sqlite", databaseFile+"?mode=ro")
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer db.Close()

		repo, err := repository.NewSQLiteRepository(db)
		if err != nil {
			log.Fatalf("init repository: %v", err)
		}

		svc := service.NewEventService(repo, repo)
		events, err := svc.ListAll(nil)
		if err != nil {
			log.Fatalf("list events: %v", err)
		}

		f, err := os.Create(*exportICS)
		if err != nil {
			log.Fatalf("create file: %v", err)
		}
		if err := ical.Encode(f, events); err != nil {
			f.Close()
			log.Fatalf("encode ical: %v", err)
		}
		if err := f.Close(); err != nil {
			log.Fatalf("close file: %v", err)
		}

		log.Printf("exported %d events to %s", len(events), *exportICS)
		return
	}

	db, err := repository.OpenDB(databaseFile, 5000,
		"mmap_size=134217728",
		"synchronous=NORMAL",
	)
	if err != nil {
		if errors.Is(err, commonsqlite.ErrSchemaTooNew) {
			log.Fatalf("database %s was written by a newer version of mycal; upgrade the binary: %v", databaseFile, err)
		}
		if errors.Is(err, repository.ErrLegacySchema) {
			log.Fatalf("database %s: %v", databaseFile, err)
		}
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err = ensureWritable(db); err != nil {
		log.Fatalf("open database: %v", err)
	}

	// One-shot: update query planner statistics for all connections.
	if _, err := db.Exec("PRAGMA optimize"); err != nil {
		log.Fatalf("PRAGMA optimize: %v", err)
	}

	repo, err := repository.NewSQLiteRepository(db)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	calSvc := service.NewCalendarService(repo)
	svc := service.NewEventService(repo, repo)
	prefSvc := service.NewPreferencesService(repo)
	feedSvc := service.NewFeedService(repo, repo, repo)
	apiRouter := handler.NewRouter(svc, prefSvc, feedSvc, calSvc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background feed refresh goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				feedSvc.RefreshAllDue()
			}
		}
	}()

	resolvedMymailURL := deriveMymailURL(*publicURL)
	if resolvedMymailURL != "" {
		log.Printf("mycal: MyMail URL configured as %s", resolvedMymailURL)
	}
	resolvedMynotesURL := deriveMynotesURL(*publicURL)
	if resolvedMynotesURL != "" {
		log.Printf("mycal: MyNotes URL configured as %s", resolvedMynotesURL)
	}

	importMapHash, err := commonweb.ImportMapCSPHash(web.Static)
	if err != nil {
		log.Fatalf("compute importmap CSP hash: %v", err)
	}

	cfg := serverConfig{MymailURL: resolvedMymailURL, MynotesURL: resolvedMynotesURL}
	indexHTML, configScriptHash, err := buildIndexHTML(web.Static, "/", cfg)
	if err != nil {
		log.Fatalf("build index.html: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", apiRouter)
	mux.Handle("GET /calendar.ics", apiRouter)
	if err := mountStatic(mux, indexHTML); err != nil {
		log.Fatalf("%v", err)
	}

	// importMapHash is the 'sha256-…' CSP token for the inline importmap in
	// index.html (see web.ImportMapCSPHash); it is added to script-src so the
	// importmap is allowed without 'unsafe-inline'. configScriptHash is the
	// 'sha256-…' token for the injected server-config inline script (empty if
	// no script was injected).
	scriptSrc := "'self' " + importMapHash
	if configScriptHash != "" {
		scriptSrc += " " + configScriptHash
	}
	opts.csp = contentSecurityPolicy(scriptSrc)

	if err := serveHTTP(mux, opts, cancel); err != nil {
		log.Fatalf("%v", err)
	}
}

// httpServerOptions carries the deployment-level settings shared by the real
// server and the demo server.
type httpServerOptions struct {
	addr           string
	port           int
	publicURL      string
	basicAuthFile  string
	basicAuthRealm string
	// httpsMode makes the server emit Strict-Transport-Security; set it when the
	// app is served over HTTPS, directly or via a TLS-terminating proxy.
	httpsMode bool
	csp       string
}

// mountStatic serves the embedded static assets on mux, answering "/" and
// "/index.html" with the prepared (config-injected) SPA shell.
func mountStatic(mux *http.ServeMux, indexHTML []byte) error {
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		return fmt.Errorf("static fs: %w", err)
	}
	staticHandler, err := httputil.StaticHandler(staticFS)
	if err != nil {
		return fmt.Errorf("static handler: %w", err)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write(indexHTML)
			return
		}
		staticHandler.ServeHTTP(w, r)
	})
	return nil
}

// serveHTTP wraps mux in the standard middleware chain — CSRF, security
// headers, optional basic auth, global request body limit — and serves it until
// the process is signalled to stop. onShutdown, when non-nil, runs before the
// graceful shutdown begins.
func serveHTTP(mux http.Handler, opts httpServerOptions, onShutdown func()) error {
	serverOrigin, err := csrf.ResolveServerOrigin(opts.publicURL, opts.addr, opts.port)
	if err != nil {
		return err
	}
	httpHandler := csrf.Middleware(serverOrigin)(mux)

	hsts := ""
	if opts.httpsMode {
		hsts = "max-age=31536000; includeSubDomains"
	}
	httpHandler = httputil.SecurityHeaders(httputil.SecurityHeadersOptions{
		CSP:            opts.csp,
		ReferrerPolicy: "strict-origin-when-cross-origin",
		HSTS:           hsts,
	})(httpHandler)
	if opts.basicAuthFile != "" {
		// Strict: every non-blank line must be a "username:bcrypt-hash" pair, so a
		// login the operator believes in cannot silently not exist. Usernames carry
		// no meaning of their own here — they name no file, table or path — so no
		// username validator is passed.
		htpasswd, err := auth.LoadHtpasswdStrict(opts.basicAuthFile, nil)
		if err != nil {
			return fmt.Errorf("load htpasswd: %w", err)
		}
		httpHandler = htpasswd.Middleware(opts.basicAuthRealm)(httpHandler)
		log.Printf("basic authentication enabled")
	}
	httpHandler = http.MaxBytesHandler(httpHandler, 10*1024*1024) // 10 MiB global request body limit (matches import endpoint)

	serverAddr := fmt.Sprintf("%s:%d", opts.addr, opts.port)
	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       time.Minute,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		signal.Stop(sigCh)
		log.Println("shutting down...")
		if onShutdown != nil {
			onShutdown()
		}
		shutdownCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		srv.Shutdown(shutdownCtx) //nolint:errcheck
	}()

	log.Printf("Starting server on %s", serverAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// runDemoServer serves the backend-less demo: the ordinary web UI plus a
// service worker (web/static/demo-sw.js) that answers the REST API from
// browser-local storage. No database is opened, no feeds are refreshed, and no
// sibling apps are assumed — a demo has no server to relay anything to.
func runDemoServer(opts httpServerOptions) error {
	importMapHash, err := commonweb.ImportMapCSPHash(web.Static)
	if err != nil {
		return fmt.Errorf("compute importmap CSP hash: %w", err)
	}
	indexHTML, configScriptHash, err := buildIndexHTML(web.Static, "/", serverConfig{Demo: true})
	if err != nil {
		return fmt.Errorf("build index.html: %w", err)
	}
	opts.csp = contentSecurityPolicy("'self' " + importMapHash + " " + configScriptHash)

	mux := http.NewServeMux()
	// Nothing here answers the REST API — the service worker does. Claiming the
	// prefix anyway means a request that escapes the worker fails as an API
	// error rather than falling through to the static handler, where the client
	// would try to parse a page of HTML as JSON.
	mux.HandleFunc("/api/v1/", demoAPIUnavailable)
	if err := mountStatic(mux, indexHTML); err != nil {
		return err
	}

	log.Printf("demo mode: no database; the browser stores all data locally")
	return serveHTTP(mux, opts, nil)
}

func printVersion() {
	fmt.Println("MyCal")
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	settings := make(map[string]string, len(info.Settings))
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if vcs, ok := settings["vcs"]; ok {
		fmt.Printf("%s ", vcs)
	}
	modified := settings["vcs.modified"] == "true"
	if rev, ok := settings["vcs.revision"]; ok {
		if modified {
			fmt.Printf("revision: %s (dirty)\n", rev)
		} else {
			fmt.Printf("revision: %s\n", rev)
		}
	}
	if t, ok := settings["vcs.time"]; ok {
		if parsedTime, err := time.Parse(time.RFC3339, t); err == nil {
			fmt.Printf("updated at: %s\n", parsedTime.Local().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("updated at: %s\n", t)
		}
	}
}

func ensureWritable(db *sql.DB) error {
	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer func(conn *sql.Conn) {
		_ = conn.Close()
	}(conn)

	return conn.Raw(func(c any) error {
		if d, ok := c.(interface{ IsReadOnly(string) (bool, error) }); ok {
			// Use "main" for the primary database schema
			isReadOnly, err := d.IsReadOnly("main")
			if err != nil {
				return err
			}
			if isReadOnly {
				return fmt.Errorf("database is read-only")
			}
			return nil
		}

		return fmt.Errorf("cannot check if database is read-only")
	})
}
