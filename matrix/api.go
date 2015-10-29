package matrix

type Client interface {
	SendText(roomID, text string) error
	SendImage(roomID, text string, image *Image) error
	SendEmote(matrixRoom, emote string) error
	JoinRoom(roomID string) error
	ListRooms() (map[string]bool, error)
	GetRoomMembers(roomID string) (map[string]UserInfo, error)
	Invite(roomID, userID string) error

	Homeserver() string
	AccessToken() string
}

type Image struct {
	URL  string
	Info *ImageInfo
}

type UserInfo struct {
	AvatarURL   string `json:"avatar_url"`
	DisplayName string `json:"displayname"`
	Membership  string `json:"membership"`
}
