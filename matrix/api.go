package matrix

type Client interface {
	SendText(roomID, text string) error
	JoinRoom(roomID string) error

	Homeserver() string
	AccessToken() string
}
