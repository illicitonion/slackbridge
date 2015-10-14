package bridge

import (
	"reflect"
	"testing"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"
)

func TestSlackMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	users := NewUserMap()
	matrixUser := &matrix.User{"@nancy:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	rooms := NewRoomMap()
	rooms.Link("!abc123:matrix.org", "CANTINA")

	bridge := Bridge{users, rooms}
	bridge.OnSlackMessage(slack.Message{
		Type:    "message",
		Channel: "CANTINA",
		User:    "U34",
		Text:    "Take more chances",
	})

	want := []call{call{"SendText", []interface{}{"!abc123:matrix.org", "Take more chances"}}}
	if !reflect.DeepEqual(mockMatrixClient.calls, want) {
		t.Fatalf("Wrong Matrix calls, want %v got %v", want, mockMatrixClient.calls)
	}
}

func TestMatrixMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	users := NewUserMap()
	matrixUser := &matrix.User{"@sean:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U35", mockSlackClient}
	users.Link(matrixUser, slackUser)

	rooms := NewRoomMap()
	rooms.Link("!abc123:matrix.org", "BOWLINGALLEY")

	bridge := Bridge{users, rooms}
	bridge.OnMatrixRoomMessage(matrix.RoomMessage{
		Type:    "m.room.message",
		Content: []byte(`{"msgtype": "m.text", "body": "It's Nancy!"}`),
		UserID:  "@sean:st.andrews",
		RoomID:  "!abc123:matrix.org",
	})

	want := []call{call{"SendText", []interface{}{"BOWLINGALLEY", "It's Nancy!"}}}
	if !reflect.DeepEqual(mockSlackClient.calls, want) {
		t.Fatalf("Wrong Slack calls, want %v got %v", want, mockSlackClient.calls)
	}
}

type call struct {
	method string
	args   []interface{}
}

type MockMatrixClient struct {
	calls []call
}

func (m *MockMatrixClient) SendText(roomID, text string) error {
	m.calls = append(m.calls, call{"SendText", []interface{}{roomID, text}})
	return nil
}

type MockSlackClient struct {
	calls []call
}

func (m *MockSlackClient) SendText(channelID, text string) error {
	m.calls = append(m.calls, call{"SendText", []interface{}{channelID, text}})
	return nil
}
