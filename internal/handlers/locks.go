package handlers

import "sync"

// Every update is handled on its own goroutine so that one slow PDF does
// not hold up other parents. That makes two updates from the *same* person
// concurrent, which is a problem: each handler reads the conversation
// state, changes it, and writes it back. Two overlapping updates can read
// the same state and one write then overwrites the other.
//
// It happens in practice — a double tap on Send, or an album of photos
// arriving as several updates at once — and the consequences are real: the
// same appeal filed twice, or a photo silently dropped.
//
// userLocks serialises updates per person while leaving different people
// fully parallel.
type userLocks struct {
	mu    sync.Mutex
	locks map[int64]*lockEntry
}

type lockEntry struct {
	mu      sync.Mutex
	waiting int
}

func newUserLocks() *userLocks {
	return &userLocks{locks: make(map[int64]*lockEntry)}
}

// lock blocks until this person's previous update has finished, and returns
// the function that releases it.
func (l *userLocks) lock(userID int64) func() {
	l.mu.Lock()
	entry, ok := l.locks[userID]
	if !ok {
		entry = &lockEntry{}
		l.locks[userID] = entry
	}
	entry.waiting++
	l.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		// Drop the entry once nobody is waiting on it, so the map does not
		// grow by one entry per person forever.
		l.mu.Lock()
		entry.waiting--
		if entry.waiting == 0 {
			delete(l.locks, userID)
		}
		l.mu.Unlock()
	}
}

// updateLocks is the process-wide instance used by HandleUpdate.
var updateLocks = newUserLocks()
