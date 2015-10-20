package matrix

type Client interface {
	SendText(roomID, text string) error

	Homeserver() string
	AccessToken() string
}
