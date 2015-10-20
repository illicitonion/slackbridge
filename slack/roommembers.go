package slack

type RoomMembers struct {
	Members map[string][]*User
}

func (m *RoomMembers) Any(channel string) *User {
	users, ok := m.Members[channel]
	if !ok || len(users) == 0 {
		return nil
	}
	return users[0]
}
