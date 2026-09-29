package routes

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tertua/tupay/platform/database"
)

// renderOutboxMetrics renders outbox queue gauges for the scrape endpoint; empty on any failure so a scrape never breaks.
func renderOutboxMetrics() string {
	db, err := database.OpenDBConnection()
	if err != nil {
		return ""
	}
	stats, err := db.OutboxStats()
	if err != nil || len(stats) == 0 {
		return ""
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Queue != stats[j].Queue {
			return stats[i].Queue < stats[j].Queue
		}
		return stats[i].Status < stats[j].Status
	})
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP tupay_outbox_queue_depth Outbox queue rows by status.\n")
	fmt.Fprintf(&b, "# TYPE tupay_outbox_queue_depth gauge\n")
	for _, s := range stats {
		fmt.Fprintf(&b, "tupay_outbox_queue_depth{queue=%q,status=%q} %d\n", s.Queue, s.Status, s.Count)
	}
	fmt.Fprintf(&b, "# HELP tupay_outbox_oldest_seconds Age of the oldest outbox row in seconds.\n")
	fmt.Fprintf(&b, "# TYPE tupay_outbox_oldest_seconds gauge\n")
	for _, s := range stats {
		if s.Oldest == nil {
			continue
		}
		fmt.Fprintf(&b, "tupay_outbox_oldest_seconds{queue=%q,status=%q} %d\n", s.Queue, s.Status, int64(time.Since(*s.Oldest).Seconds()))
	}
	return b.String()
}
