// Command server is the HTTP skeleton for the assistant: a health endpoint plus
// an optional -migrate flag to apply DB migrations and exit.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/apolaki/solar-assistant/internal/config"
	"github.com/apolaki/solar-assistant/internal/db"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}
	if *migrateOnly {
		log.Println("migrations applied")
		os.Exit(0)
	}

	http.HandleFunc("/assistant/health", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{"status": "ok"}
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "db_unreachable"
		}
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.Printf("encode health: %v", err)
		}
	})
	log.Println("listening on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
