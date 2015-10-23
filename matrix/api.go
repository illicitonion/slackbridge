package matrix

type Client interface {
	SendText(roomID, text string) error
	SendImage(roomID, text string, image *Image) error
	JoinRoom(roomID string) error
	ListRooms() (map[string]bool, error)

	Homeserver() string
	AccessToken() string
}

type Image struct {
	URL  string
	Info *ImageInfo
}
