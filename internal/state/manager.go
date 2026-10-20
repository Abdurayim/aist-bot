package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"aist-bot/internal/models"
)

// Manager stores where each chat is inside a conversation flow. State lives
// in SQLite so a restart never strands a parent mid-form, with an in-memory
// cache in front of it because every incoming update reads it.
type Manager struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[int64]cachedState
}

type cachedState struct {
	state string
	data  json.RawMessage
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db, cache: make(map[int64]cachedState)}
}

// Set stores the state and its collected data.
func (m *Manager) Set(telegramID int64, state string, data *models.StateData) error {
	if data == nil {
		data = &models.StateData{}
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	_, err = m.db.Exec(`
		INSERT INTO user_states (telegram_id, state, data, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (telegram_id) DO UPDATE SET
			state = excluded.state, data = excluded.data, updated_at = CURRENT_TIMESTAMP`,
		telegramID, state, string(encoded))
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	m.mu.Lock()
	m.cache[telegramID] = cachedState{state: state, data: encoded}
	m.mu.Unlock()

	return nil
}

// Get returns the current state name and its data. A chat that has never
// been seen returns an empty state name and empty data, never nil.
func (m *Manager) Get(telegramID int64) (string, *models.StateData, error) {
	m.mu.RLock()
	cached, ok := m.cache[telegramID]
	m.mu.RUnlock()

	if !ok {
		var state string
		var raw sql.NullString

		err := m.db.QueryRow(`SELECT state, data FROM user_states WHERE telegram_id = ?`, telegramID).
			Scan(&state, &raw)

		if err == sql.ErrNoRows {
			return "", &models.StateData{}, nil
		}
		if err != nil {
			return "", &models.StateData{}, fmt.Errorf("failed to read state: %w", err)
		}

		cached = cachedState{state: state, data: json.RawMessage(raw.String)}

		m.mu.Lock()
		m.cache[telegramID] = cached
		m.mu.Unlock()
	}

	data := &models.StateData{}
	if len(cached.data) > 0 {
		if err := json.Unmarshal(cached.data, data); err != nil {
			// Corrupt state should restart the flow, not break the bot.
			return cached.state, &models.StateData{}, nil
		}
	}

	return cached.state, data, nil
}

// Clear drops the conversation state of a chat.
func (m *Manager) Clear(telegramID int64) error {
	if _, err := m.db.Exec(`DELETE FROM user_states WHERE telegram_id = ?`, telegramID); err != nil {
		return fmt.Errorf("failed to clear state: %w", err)
	}

	m.mu.Lock()
	delete(m.cache, telegramID)
	m.mu.Unlock()

	return nil
}

// CleanOlderThan removes abandoned states, so a half-filled form does not
// linger for weeks. It also drops them from the cache.
func (m *Manager) CleanOlderThan(hours int) error {
	_, err := m.db.Exec(
		fmt.Sprintf(`DELETE FROM user_states WHERE updated_at < datetime('now', '-%d hours')`, hours),
	)
	if err != nil {
		return fmt.Errorf("failed to clean states: %w", err)
	}

	m.mu.Lock()
	m.cache = make(map[int64]cachedState)
	m.mu.Unlock()

	return nil
}
