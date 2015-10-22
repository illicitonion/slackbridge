package bridge

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

type Config struct {
	MatrixASAccessToken string
	UserPrefix          string
	HomeserverBaseURL   string
	HomeserverName      string
}

type Bridge struct {
	UserMap          *UserMap
	RoomMap          *RoomMap
	SlackRoomMembers *slack.RoomMembers
	Client           http.Client
	EchoSuppresser   *matrix.EchoSuppresser
	Config           Config
}

func (b *Bridge) OnSlackMessage(m slack.Message) {
	matrixRoom := b.RoomMap.MatrixForSlack(m.Channel)
	if matrixRoom == "" {
		log.Printf("Ignoring event for unknown slack room %q", m.Channel)
		return
	}
	matrixUser := b.UserMap.MatrixForSlack(m.User)
	if matrixUser == nil {
		matrixUser = b.matrixUserFor(m.Channel, m.User, matrixRoom)
	}
	if matrixUser == nil {
		log.Printf("Ignoring event from unknown slack user %q", m.User)
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
	if slackUser == nil {
		log.Printf("Ignoring event from unknown matrix user %q", m.UserID)
		return
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

func (b *Bridge) slackUserFor(slackChannel, matrixUserID string) *slack.User {
	token := b.botAccessToken(slackChannel)
	if token == "" {
		return nil
	}
	client := slack.NewBotClient(token, matrixUserID, b.Client, b.RoomMap.ShouldNotify)
	user := &slack.User{matrixUserID, client}
	b.SlackRoomMembers.Add(slackChannel, user)
	return user
}

func (b *Bridge) botAccessToken(slackChannel string) string {
	user := b.SlackRoomMembers.Any(slackChannel)
	if user == nil {
		return ""
	}
	return user.Client.AccessToken()
}

func (b *Bridge) matrixUserFor(slackChannel, slackUserID, matrixRoomID string) *matrix.User {
	user := b.SlackRoomMembers.Any(slackChannel)
	if user == nil {
		return nil
	}
	resp, err := b.Client.Get(fmt.Sprintf("https://slack.com/api/users.info?token=%s&user=%s", user.Client.AccessToken(), slackUserID))
	if err != nil {
		log.Printf("Error looking up user %q: %v", slackUserID, err)
		return nil
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading user info response: %v", err)
		return nil
	}
	var r slackUserInfoResponse
	if err := json.Unmarshal(respBytes, &r); err != nil {
		log.Printf("Error unmarshaling user info response: %v (%s)", err, string(respBytes))
		return nil
	}
	matrixUserID := b.Config.UserPrefix + r.User.Name + ":" + b.Config.HomeserverName
	client := matrix.NewBotClient(b.Config.MatrixASAccessToken, matrixUserID, b.Client, b.Config.HomeserverBaseURL, b.EchoSuppresser)
	if err := client.JoinRoom(matrixRoomID); err != nil {
		log.Printf("Error joining room: %v", err)
		return nil
	}
	return &matrix.User{matrixUserID, client}
}

type slackUserInfoResponse struct {
	OK   bool       `json:"ok"`
	User *slackUser `json:"user"`
}

type slackUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
