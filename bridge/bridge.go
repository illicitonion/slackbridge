package bridge

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

type Bridge struct {
	UserMap          *UserMap
	RoomMap          *RoomMap
	SlackRoomMembers *slack.RoomMembers
	Client           http.Client
}

func (b *Bridge) OnSlackMessage(m slack.Message) {
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
	if err := matrixUser.Client.SendText(matrixRoom, slackToMatrix(m.Text)); err != nil {
		log.Printf("Error sending text to Matrix: %v", err)
	}
}

func (b *Bridge) OnMatrixRoomMessage(m matrix.RoomMessage) {
	slackChannel := b.RoomMap.SlackForMatrix(m.RoomID)
	if slackChannel == "" {
		log.Printf("Ignoring event for unknown matrix room %q", m.RoomID)
		return
	}
	slackUser := b.UserMap.SlackForMatrix(m.UserID)
	if slackUser == nil {
		slackUser = b.slackUserFor(slackChannel, m.UserID)
	}
	var c matrix.TextMessageContent
	if err := json.Unmarshal(m.Content, &c); err != nil {
		log.Printf("Error unmarshaling room message content: %v", err)
		return
	}
	if err := slackUser.Client.SendText(slackChannel, matrixToSlack(c.Body)); err != nil {
		log.Printf("Error sending text to Slack: %v", err)
	}
}

func (b *Bridge) slackUserFor(slackChannel, userID string) *slack.User {
	token := b.botAccessToken(slackChannel)
	client := slack.NewBotClient(token, userID, b.Client, b.RoomMap)
	return &slack.User{userID, client}
}

func (b *Bridge) botAccessToken(slackChannel string) string {
	user := b.SlackRoomMembers.Any(slackChannel)
	if user == nil {
		return ""
	}
	return user.Client.AccessToken()
}
