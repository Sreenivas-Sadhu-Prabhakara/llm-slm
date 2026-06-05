// Command server runs the Apolaki solar-assistant HTTP service: the streaming
// chat API, feedback hook, health endpoint, and local test page. The -migrate
// flag applies DB migrations and exits.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/apolaki/solar-assistant/internal/chatlog"
	"github.com/apolaki/solar-assistant/internal/config"
	"github.com/apolaki/solar-assistant/internal/db"
	"github.com/apolaki/solar-assistant/internal/embed"
	"github.com/apolaki/solar-assistant/internal/generator"
	"github.com/apolaki/solar-assistant/internal/httpapi"
	"github.com/apolaki/solar-assistant/internal/personalizer"
	"github.com/apolaki/solar-assistant/internal/retriever"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations and exit")
	addr := flag.String("addr", ":8090", "listen address")
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

	emb := embed.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.EmbedModel)
	handler := httpapi.Handler(httpapi.Deps{
		Retriever:    retriever.New(pool, emb),
		Generator:    generator.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.GenModel),
		Personalizer: personalizer.Noop{},
		Logger:       chatlog.New(pool),
		Pool:         pool,
		GenModel:     cfg.GenModel,
		JWTSecret:    cfg.JWTSecret,
	})

	log.Printf("apolaki solar-assistant listening on %s (chat UI at /)", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
