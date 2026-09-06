// institut-ingester is ONE microservice per data source (INSTITUT=ifop, ipsos,
// elabe, opinionway, harris, odoxa, cluster17, verian, or commission). Each
// scrapes its source's own site — no Wikipedia, no dead third-party aggregator
// (nsppolls stopped at 2022) — and publishes poll.raw. Deploy N copies, one per
// INSTITUT, so a layout change on one site only breaks that one service.
package main

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/ggalperine/election2027/internal/config"
	"github.com/ggalperine/election2027/internal/poll"
	"github.com/ggalperine/election2027/internal/rabbit"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	key := os.Getenv("INSTITUT")
	if key == "" {
		log.Fatal().Msg("INSTITUT env is required (e.g. ifop, elabe, commission)")
	}
	src, ok := poll.SourceByKey(key)
	if !ok {
		log.Fatal().Str("institut", key).Msg("unknown INSTITUT")
	}

	rb, err := rabbit.Dial(cfg.AMQPURL)
	if err != nil {
		log.Fatal().Err(err).Msg("rabbit")
	}
	defer rb.Close()

	log.Info().Str("source", src.Name()).Msg("institut-ingester started")
	fetchAndPublish(ctx, rb, src)

	// Each institut has its OWN queue bound to the shared fetch trigger, so the
	// scheduler's daily cron fans out to every institut at once.
	deliveries, err := rb.Consume("institut-"+key, rabbit.KeyFetchPolls)
	if err != nil {
		log.Fatal().Err(err).Msg("consume")
	}
	for d := range deliveries {
		fetchAndPublish(ctx, rb, src)
		_ = d.Ack(false)
	}
}

func fetchAndPublish(ctx context.Context, rb *rabbit.Conn, src poll.Source) {
	polls, err := src.Fetch(ctx)
	if err != nil {
		log.Error().Err(err).Str("source", src.Name()).Msg("fetch")
		return
	}
	var published int
	for _, p := range polls {
		if len(p.Results) == 0 {
			continue // metadata-only (e.g. Commission index without %): skip until enriched
		}
		if err := rb.Publish(ctx, rabbit.KeyPollRaw, p); err != nil {
			log.Error().Err(err).Msg("publish poll")
			continue
		}
		published++
	}
	log.Info().Str("source", src.Name()).Int("fetched", len(polls)).Int("published", published).Msg("done")
}
