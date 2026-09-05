// results-ingester fetches official election results (interieur.gouv.fr open
// data) and publishes geo.result messages. Triggered by cmd.fetch.geo.
package main

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/ggalperine/election2027/internal/config"
	"github.com/ggalperine/election2027/internal/rabbit"
	"github.com/ggalperine/election2027/internal/results"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	rb, err := rabbit.Dial(cfg.AMQPURL)
	if err != nil {
		log.Fatal().Err(err).Msg("rabbit")
	}
	defer rb.Close()

	src := chooseSource()
	log.Info().Str("source", src.Name()).Msg("results-ingester started")

	ingest(ctx, rb, src)

	deliveries, err := rb.Consume("results-ingester", rabbit.KeyFetchGeo)
	if err != nil {
		log.Fatal().Err(err).Msg("consume")
	}
	for d := range deliveries {
		log.Info().Msg("fetch trigger received")
		ingest(ctx, rb, src)
		_ = d.Ack(false)
	}
}

func chooseSource() results.Source {
	// GOV_CSV_URL points at an open-data results export from data.gouv.fr.
	if url := os.Getenv("GOV_CSV_URL"); url != "" {
		return results.CSVSource{URL: url}
	}
	return results.SampleSource{Election: "presidentielle-2022-t1"}
}

func ingest(ctx context.Context, rb *rabbit.Conn, src results.Source) {
	rows, err := src.Fetch(ctx)
	if err != nil {
		log.Error().Err(err).Msg("fetch geo")
		return
	}
	for _, g := range rows {
		if err := rb.Publish(ctx, rabbit.KeyGeoResult, g); err != nil {
			log.Error().Err(err).Msg("publish geo")
		}
	}
	log.Info().Int("count", len(rows)).Msg("published geo results")
}
