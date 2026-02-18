package main

import (
	"database/sql"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/auth"
	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "mycal.db", "database file path")
	basicAuthFile := flag.String("basic-auth-file", "", "path to htpasswd file for HTTP basic authentication (bcrypt only)")
	flag.Parse()

	var authMiddleware func(http.Handler) http.Handler
	if *basicAuthFile != "" {
		htpasswd, err := auth.LoadHtpasswd(*basicAuthFile)
		if err != nil {
			log.Fatalf("load htpasswd: %v", err)
		}
		authMiddleware = htpasswd.Middleware
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
