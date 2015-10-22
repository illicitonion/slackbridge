package slack

import "sync"

func NewRoomMembers() *RoomMembers {
	return &RoomMembers{
		Members: make(map[string][]*User),
	}
}

type RoomMembers struct {
	mu      sync.RWMutex
	Members map[string][]*User
}

func (m *RoomMembers) Any(channel string) *User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	users, ok := m.Members[channel]
	if !ok || len(users) == 0 {
		return nil
	}
	return users[0]
}

func (m *RoomMembers) Add(channel string, user *User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Members[channel] = append(m.Members[channel], user)
}
