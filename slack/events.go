package slack

import (
	"log"
	"strconv"
)

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

func (m *Message) Timestamp() float64 {
	f, err := strconv.ParseFloat(m.TS, 64)
	if err != nil {
		log.Printf("Error parsing timestamp: %q: %v", m.TS, err)
		return -1
	}
	return f
}
