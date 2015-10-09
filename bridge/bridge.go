package bridge

import (
	"log"

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
