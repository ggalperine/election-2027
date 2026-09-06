package poll

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestCommissionLive hits the real Commission des sondages site + notice PDFs.
// Guarded: runs only with LIVE=1 and requires pdftotext on PATH. It is a smoke
// test for the full voie-A pipeline (index → PDF → pdftotext → parsed poll).
func TestCommissionLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run the live Commission integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	c := Commission{Cycle: "2027", MaxNotices: 4}
	polls, err := c.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(polls) == 0 {
		t.Fatalf("no polls parsed from the live index")
	}
	for _, p := range polls {
		t.Logf("%s | %s | n=%d | %s | %d candidates | %s",
			p.FieldEnd.Format("2006-01-02"), p.Pollster, p.SampleSize,
			p.ExternalID, len(p.Results), p.SourceURL)
		var sum float64
		for _, v := range p.Results {
			sum += v
		}
		if sum < 85 || sum > 115 {
			t.Errorf("poll %s: implausible total %.1f", p.ExternalID, sum)
		}
	}
}
