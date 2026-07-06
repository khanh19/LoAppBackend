package lists

import "encore.dev/cron"

var _ = cron.NewJob("sync-place-lists", cron.JobConfig{
	Title:    "Sync curated place lists from seed JSON",
	Every:    24 * cron.Hour,
	Endpoint: SyncPlaceLists,
})
