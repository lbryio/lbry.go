package cmd

const defaultMaxTries = 3

var (
	stopOnError             bool
	maxTries                int
	takeOverExistingChannel bool
	refill                  int
	limit                   int
)

const (
	StatusPending = "pending" // waiting for permission to sync
	StatusQueued  = "queued"  // in sync queue. will be synced soon
	StatusSyncing = "syncing" // syncing now
	StatusSynced  = "synced"  // done
)

var SyncStatuses = []string{StatusPending, StatusQueued, StatusSyncing, StatusSynced}
