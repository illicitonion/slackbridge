package matrix

import "encoding/json"

type RoomMessage struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
	UserID  string          `json:"user_id"`
	RoomID  string          `json:"room_id"`
	EventID string          `json:"event_id"`
}

type TextMessageContent struct {
	Body    string `json:"body"`
	MsgType string `json:"msgtype"`
}

type ImageMessageContent struct {
	Body    string     `json:"body"`
	MsgType string     `json:"msgtype"`
	URL     string     `json:"url"`
	Info    *ImageInfo `json:"info"`
}

type ImageInfo struct {
	Height   int    `json:"h"`
	Width    int    `json:"w"`
	MIMEType string `json:"mimetype"`
	Size     int64  `json:"size"`
}
