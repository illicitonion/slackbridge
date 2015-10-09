package slack

type event struct {
	Type string
}

type Hello struct {
	Type string `json:"type"`
}

type Message struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	User    string `json:"user"`
	Text    string `json:"text"`
}
