// aggregator consumes poll.raw + geo.result messages, persists them, and
// recomputes rolling poll averages after each batch.
package main

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"

	"github.com/ggalperine/election2027/internal/config"
	"github.com/ggalperine/election2027/internal/models"
	"github.com/ggalperine/election2027/internal/rabbit"
	"github.com/ggalperine/election2027/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("store")
	}
	defer st.Close()

	rb, err := rabbit.Dial(cfg.AMQPURL)
	if err != nil {
		log.Fatal().Err(err).Msg("rabbit")
	}
	defer rb.Close()

	deliveries, err := rb.Consume("aggregator", rabbit.KeyPollRaw, rabbit.KeyGeoResult)
	if err != nil {
		log.Fatal().Err(err).Msg("consume")
	}
	log.Info().Msg("aggregator consuming")

	for d := range deliveries {
		switch d.RoutingKey {
		case rabbit.KeyPollRaw:
			var p models.RawPoll
			if err := json.Unmarshal(d.Body, &p); err != nil {
				log.Error().Err(err).Msg("bad poll msg")
				_ = d.Nack(false, false)
				continue
			}
			if err := st.SavePoll(ctx, p); err != nil {
				log.Error().Err(err).Msg("save poll")
				_ = d.Nack(false, true)
				continue
			}
			log.Info().Str("poll", p.ExternalID).Msg("saved poll")
			_ = d.Ack(false)

		case rabbit.KeyGeoResult:
			var g models.GeoResult
			if err := json.Unmarshal(d.Body, &g); err != nil {
				log.Error().Err(err).Msg("bad geo msg")
				_ = d.Nack(false, false)
				continue
			}
			if err := st.SaveGeoResult(ctx, g); err != nil {
				log.Error().Err(err).Msg("save geo")
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
