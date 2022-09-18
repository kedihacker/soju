// Package strintern implements string interning.
//
// String interning can be used to reduce memory usage when many identical
// strings are duplicated in memory.
//
// soju uses this mechanism to de-duplicate nicknames and channel names when
// many bouncer users share the same channels.
package strintern

import (
	"sync"
	"sync/atomic"
	"time"
)

// resetInterval is the interval at which the cache is reset.
//
// To avoid having an always-growing string cache, we reset our cache
// regularily. This should still allow mass JOIN bursts to share strings.
const resetInterval = 24 * time.Hour

var m atomic.Value // sync.Map

func init() {
	m.Store(new(sync.Map))

	go func() {
		ch := time.Tick(resetInterval)

		for range ch {
			m.Store(new(sync.Map))
		}
	}()
}

// LoadOrStore inserts the string in the cache if it's missing, returns the
// existing string otherwise.
func LoadOrStore(s string) string {
	m := m.Load().(*sync.Map)

	// TODO: figure out whether cloning the string on store is desirable here
	v, _ := m.LoadOrStore(s, s)
	return v.(string)
}
