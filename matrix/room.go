package matrix

func NewRoom(id string) *Room {
	return &Room{
		ID:    id,
		Users: make(map[string]UserInfo),
	}
}

type Room struct {
	ID              string
	LastStreamToken string
	Users           map[string]UserInfo
}
