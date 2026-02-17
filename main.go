package main

import (
	"database/sql"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "mycal.db", "database file path")
	flag.Parse()

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

	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
