// scheduler runs the CRON job that triggers ingesters every 24h by publishing
// command messages to RabbitMQ.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/ggalperine/election2027/internal/config"
	"github.com/ggalperine/election2027/internal/rabbit"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	rb, err := rabbit.Dial(cfg.AMQPURL)
	if err != nil {
		log.Fatal().Err(err).Msg("rabbit")
	}
	defer rb.Close()

	// CRON_SPEC overrides the default (daily at 06:00). Format: robfig/cron.
	spec := os.Getenv("CRON_SPEC")
	if spec == "" {
		spec = "0 6 * * *" // every day 06:00
	}

	c := cron.New()
	_, err = c.AddFunc(spec, func() {
		log.Info().Msg("cron: triggering daily fetch")
		if err := rb.Publish(ctx, rabbit.KeyFetchPolls, map[string]string{"reason": "cron"}); err != nil {
			log.Error().Err(err).Msg("publish fetch.polls")
		}
		if err := rb.Publish(ctx, rabbit.KeyFetchGeo, map[string]string{"reason": "cron"}); err != nil {
			log.Error().Err(err).Msg("publish fetch.geo")
		}
	})
	if err != nil {
		log.Fatal().Err(err).Msg("cron add")
	}

	c.Start()
	log.Info().Str("spec", spec).Msg("scheduler started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	c.Stop()
}
