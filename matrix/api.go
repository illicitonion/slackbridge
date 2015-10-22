package matrix

type Client interface {
	SendText(roomID, text string) error
	JoinRoom(roomID string) error
	ListRooms() (map[string]bool, error)

	Homeserver() string
	AccessToken() string
}
