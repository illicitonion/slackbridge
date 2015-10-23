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
	Subtype string `json:"subtype"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	User    string `json:"user"`
	Text    string `json:"text"`

	File *File `json:"file"`
}

func (m *Message) Timestamp() float64 {
	f, err := strconv.ParseFloat(m.TS, 64)
	if err != nil {
		log.Printf("Error parsing timestamp: %q: %v", m.TS, err)
		return -1
	}
	return f
}

type File struct {
	MIMEType       string   `json:"mimetype"`
	URL            string   `json:"url"`
	OriginalHeight int      `json:"original_h"`
	OriginalWidth  int      `json:"original_w"`
	Size           int64    `json:"size"`
	CommentsCount  int      `json:"comments_count"`
	InitialComment *Comment `json:"initial_comment"`
}

type Comment struct {
	Comment string `json:"comment"`
	User    string `json:"user"`
}
