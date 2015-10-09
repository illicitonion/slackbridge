package matrix

type TextMessage struct {
	Content *TextMessageContent `json:"content"`
	RoomID  string              `json:"room_id"`
	Type    string              `json:"type"`
	UserID  string              `json:"user-id"`
}

type TextMessageContent struct {
	Body    string `json:"body"`
	MsgType string `json:"msgtype"`
}
