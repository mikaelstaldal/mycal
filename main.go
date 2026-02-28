package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mikaelstaldal/mycal/internal/auth"
	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/web"
)

const databaseName = "mycal.sqlite"

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	addr := flag.String("addr", "", "address to listen on")
	dataDir := flag.String("data", "data", "directory to store data in")
	basicAuthFile := flag.String("basic-auth-file", "", "enable HTTP basic auth with username and password from given file in htpasswd format (bcrypt only)")
	basicAuthRealm := flag.String("basic-auth-realm", "mycal", "realm for HTTP basic auth")
	exportICS := flag.String("export-ics", "", "export all events to an .ics file and exit")
	flag.Parse()

	if *port < 1 || *port > 65535 {
		log.Fatalf("Invalid port number: %d. Must be between 1 and 65535", *port)
	}

	info, err := os.Stat(*dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(*dataDir, 0755); err != nil {
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

	db, err := sql.Open("sqlite", databaseFile)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	err = ensureWritable(db)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

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

	serverAddr := fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("Starting server on %s", serverAddr)
	server := http.Server{
		Addr:         serverAddr,
		Handler:      root,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  time.Minute,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
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
