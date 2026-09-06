// poll-ingester fetches polls from the configured consolidated sources and
// publishes poll.raw messages. Sources are selected via POLL_SOURCES
// (comma-separated): nsppolls (2022 real, all instituts), wiki2027 (2027 real,
// all instituts), sample (offline demo). Default: "nsppolls,wiki2027".
package main

import (
	"context"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ggalperine/election2027/internal/config"
	"github.com/ggalperine/election2027/internal/poll"
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

	sources := chooseSources()
	names := make([]string, len(sources))
	for i, s := range sources {
		names[i] = s.Name()
	}
	log.Info().Strs("sources", names).Msg("poll-ingester started")

	ingest(ctx, rb, sources)

	deliveries, err := rb.Consume("poll-ingester", rabbit.KeyFetchPolls)
	if err != nil {
		log.Fatal().Err(err).Msg("consume")
	}
	for d := range deliveries {
		log.Info().Msg("fetch trigger received")
		ingest(ctx, rb, sources)
		_ = d.Ack(false)
	}
}

func chooseSources() []poll.Source {
	spec := os.Getenv("POLL_SOURCES")
	if spec == "" {
		spec = "nsppolls,wiki2027"
	}
	var out []poll.Source
	for _, name := range strings.Split(spec, ",") {
		switch strings.TrimSpace(name) {
		case "nsppolls":
			out = append(out, poll.NSPPolls{Cycle: "2022"})
		case "wiki2027":
			// Commission notices own the first round; Wikipedia is kept only for
			// the second-round duels until the notice parser covers round 2.
			out = append(out, poll.Wiki2027{Round2Only: true})
		case "sample":
			out = append(out, poll.SampleSource{})
		}
	}
	if len(out) == 0 {
		out = append(out, poll.SampleSource{})
	}
	return out
}

func ingest(ctx context.Context, rb *rabbit.Conn, sources []poll.Source) {
	for _, src := range sources {
		polls, err := src.Fetch(ctx)
		if err != nil {
			log.Error().Err(err).Str("source", src.Name()).Msg("fetch polls")
			continue
		}
		for _, p := range polls {
			if err := rb.Publish(ctx, rabbit.KeyPollRaw, p); err != nil {
				log.Error().Err(err).Msg("publish poll")
			}
		}
		log.Info().Str("source", src.Name()).Int("count", len(polls)).Msg("published polls")
	}
}
