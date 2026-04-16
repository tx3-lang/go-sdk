package facade

import "time"

// PollConfig controls the polling behaviour of WaitForConfirmed and WaitForFinalized.
type PollConfig struct {
	Attempts uint          // Maximum number of status checks (default: 20)
	Delay    time.Duration // Delay between status checks (default: 5s)
}

// DefaultPollConfig returns a PollConfig with sensible defaults:
// 20 attempts with a 5-second delay between each.
func DefaultPollConfig() PollConfig {
	return PollConfig{
		Attempts: 20,
		Delay:    5 * time.Second,
	}
}
