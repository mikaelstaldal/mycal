package main

import (
	"database/sql"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/mikaelstaldal/mycal/internal/auth"
	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "mycal.db", "database file path")
	exportICS := flag.String("export-ics", "", "export all events to an .ics file and exit")
	basicAuthFile := flag.String("basic-auth-file", "", "path to htpasswd file for HTTP basic authentication (bcrypt only)")
	basicAuthRealm := flag.String("basic-auth-realm", "mycal", "HTTP basic auth realm")
	flag.Parse()

	if *exportICS != "" {
		// Open database read-only so this can run concurrently with a server
		db, err := sql.Open("sqlite", *dbPath+"?mode=ro")
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer db.Close()

		repo, err := repository.NewSQLiteRepository(db)
		if err != nil {
			log.Fatalf("init repository: %v", err)
		}

		svc := service.NewEventService(repo)
		events, err := svc.ListAll()
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

	var authMiddleware func(http.Handler) http.Handler
	if *basicAuthFile != "" {
		htpasswd, err := auth.LoadHtpasswd(*basicAuthFile)
		if err != nil {
			log.Fatalf("load htpasswd: %v", err)
		}
		authMiddleware = htpasswd.Middleware(*basicAuthRealm)
		log.Printf("basic authentication enabled")
	}

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repo, err := repository.NewSQLiteRepository(db)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	svc := service.NewEventService(repo)
	apiRouter := handler.NewRouter(svc)

	calendarFeed := handler.NewCalendarFeedHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("GET /calendar.ics", calendarFeed)

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	var root http.Handler = mux
	if authMiddleware != nil {
		root = authMiddleware(mux)
	}

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, root); err != nil {
		log.Fatalf("server: %v", err)
	}
}
