package cmd

const defaultMaxTries = 3

var (
	stopOnError             bool
	maxTries                int
	takeOverExistingChannel bool
	refill                  int
	limit                   int
	skipSpaceCheck          bool
	syncUpdate              bool
	syncStatus              string
	syncFrom                int64
	syncUntil               int64
)

const (
	StatusPending = "pending" // waiting for permission to sync
	StatusQueued  = "queued"  // in sync queue. will be synced soon
	StatusSyncing = "syncing" // syncing now
	StatusSynced  = "synced"  // done
	StatusFailed  = "failed"
)

var SyncStatuses = []string{StatusPending, StatusQueued, StatusSyncing, StatusSynced, StatusFailed}
