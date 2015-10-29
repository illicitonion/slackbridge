package matrix

func NewRoom(id string) *Room {
	return &Room{
		ID: id,
	}
}

type Room struct {
	ID              string
	LastStreamToken string
}
