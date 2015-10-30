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

type Image struct {
	URL  string
	Info *ImageInfo
}

type ImageInfo struct {
	Height   int    `json:"h"`
	Width    int    `json:"w"`
	MIMEType string `json:"mimetype"`
	Size     int64  `json:"size"`
}

type StateEvent struct {
	Content  json.RawMessage `json:"content"`
	Type     string          `json:"type"`
	StateKey string          `json:"state_key"`
}

type UserInfo struct {
	AvatarURL   string `json:"avatar_url"`
	DisplayName string `json:"displayname"`
	Membership  string `json:"membership"`
}

type RoomMemberEvent struct {
	Type     string   `json:"type"`
	StateKey string   `json:"state_key"`
	Content  UserInfo `json:"content"`
	RoomID   string   `json:"room_id"`
	UserID   string   `json:"user_id"`
}
