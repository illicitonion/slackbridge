package bridge

import (
	"encoding/json"
	"log"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

type Bridge struct {
	UserMap *UserMap
	RoomMap *RoomMap
}

func (b *Bridge) OnSlackMessage(m *slack.Message) {
	matrixUser := b.UserMap.MatrixForSlack(m.User)
	if matrixUser == nil {
		log.Printf("Ignoring event from unknown slack user %q", m.User)
		return
	}
	matrixRoom := b.RoomMap.MatrixForSlack(m.Channel)
	if matrixRoom == "" {
		log.Printf("Ignoring event for unknown slack room %q", m.Channel)
		return
	}
	if err := matrixUser.Client.SendText(matrixRoom, m.Text); err != nil {
		log.Printf("Error sending text to Matrix: %v", err)
	}
}

func (b *Bridge) OnMatrixRoomMessage(m *matrix.RoomMessage) {
	slackUser := b.UserMap.SlackForMatrix(m.UserID)
	if slackUser == nil {
		log.Printf("Ignoring event from unknown matrix user: %q", m.UserID)
		return
	}
	slackChannel := b.RoomMap.SlackForMatrix(m.RoomID)
	if slackChannel == "" {
		log.Printf("Ignoring event for unknown matrix room %q", m.RoomID)
		return
	}
	var c matrix.TextMessageContent
	if err := json.Unmarshal(m.Content, &c); err != nil {
		log.Printf("Error unmarshaling room message content: %v", err)
		return
	}
	if err := slackUser.Client.SendText(slackChannel, c.Body); err != nil {
		log.Printf("Error sending text to Slack: %v", err)
	}
}
