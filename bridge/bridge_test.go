package bridge

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matrix-org/slackbridge/matrix"
	"github.com/matrix-org/slackbridge/slack"

	_ "github.com/mattn/go-sqlite3"
)

func TestSlackMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "CANTINA")

	echoSuppresser := matrix.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := &matrix.User{"@nancy:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, http.Client{}, echoSuppresser, Config{}}
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

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "BOWLINGALLEY")

	echoSuppresser := matrix.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := &matrix.User{"@sean:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U35", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, http.Client{}, echoSuppresser, Config{}}
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

func TestMatrixMessageFromUnlinkedUser(t *testing.T) {
	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	slackChannel := "BOWLINGALLEY"
	matrixRoom := "!abc123:matrix.org"
	message := "It's Nancy!"
	matrixUser := "@sean:st.andrews"

	rooms.Link(matrixRoom, slackChannel)

	echoSuppresser := matrix.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}

	slackRoomMembers := &slack.RoomMembers{map[string][]*slack.User{
		slackChannel: []*slack.User{&slack.User{"someone", &MockSlackClient{}}},
	}}

	called := make(chan struct{}, 1)
	verify := func(req *http.Request) string {
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("Error reading request body: %v", err)
		}
		v, err := url.ParseQuery(string(b))
		if err != nil {
			t.Fatalf("Error parsing request body: %v", err)
		}
		assertUrlValueEquals(t, v, "token", "slack_access_token")
		assertUrlValueEquals(t, v, "channel", slackChannel)
		assertUrlValueEquals(t, v, "text", message)
		assertUrlValueEquals(t, v, "as_user", "false")
		assertUrlValueEquals(t, v, "username", matrixUser)
		called <- struct{}{}
		return ""
	}
	client := http.Client{
		Transport: &spyRoundTripper{verify},
	}
	bridge := Bridge{users, rooms, slackRoomMembers, client, echoSuppresser, Config{}}
	bridge.OnMatrixRoomMessage(matrix.RoomMessage{
		Type:    "m.room.message",
		Content: []byte(`{"msgtype": "m.text", "body": "` + message + `"}`),
		UserID:  matrixUser,
		RoomID:  matrixRoom,
	})

	select {
	case _ = <-called:
		return
	case _ = <-time.After(50 * time.Millisecond):
		t.Fatalf("Didn't get expected call")
	}
}

func TestSlackMessageFromUnlinkedUser(t *testing.T) {
	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	slackChannel := "BOWLINGALLEY"
	matrixRoom := "!abc123:matrix.org"
	message := "Shhhhh"
	slackUser := "U123"
	asToken := "abc123"

	rooms.Link(matrixRoom, slackChannel)

	echoSuppresser := matrix.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}

	slackRoomMembers := &slack.RoomMembers{map[string][]*slack.User{
		slackChannel: []*slack.User{&slack.User{"someone", &MockSlackClient{}}},
	}}

	called := make(chan struct{}, 1)
	didJoin := false
	verify := func(req *http.Request) string {
		if req.URL.Path == "/api/users.info" {
			return `{"ok": true, "user": {"id": "` + slackUser + `", "name": "someoneonslack"}}`
		}
		if !didJoin && req.URL.Path == "/_matrix/client/api/v1/rooms/"+matrixRoom+"/join" {
			didJoin = true
			return ""
		}
		if req.URL.Path != "/_matrix/client/api/v1/rooms/"+matrixRoom+"/send/m.room.message" {
			t.Fatalf("Got request to unexpected path %q", req.URL.Path)
			return ""
		}
		if !didJoin {
			t.Errorf("Didn't get expected join before message send")
		}

		query := req.URL.Query()
		assertUrlValueEquals(t, query, "access_token", asToken)
		assertUrlValueEquals(t, query, "user_id", "@prefix_someoneonslack:my.server")

		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("Error reading request body: %v", err)
		}
		var content matrix.TextMessageContent
		if err := json.Unmarshal(b, &content); err != nil {
			t.Fatalf("Error unmarshaling json: %v", err)
		}
		if content.MsgType != "m.text" {
			t.Errorf("Msgtype: want %q got %q", "m.text", content.MsgType)
		}
		if content.Body != message {
			t.Errorf("Message: want %q got %q", message, content.Body)
		}
		called <- struct{}{}
		return ""
	}
	client := http.Client{
		Transport: &spyRoundTripper{verify},
	}
	bridge := Bridge{users, rooms, slackRoomMembers, client, echoSuppresser, Config{
		MatrixASAccessToken: asToken,
		UserPrefix:          "@prefix_",
		HomeserverBaseURL:   "https://my.server",
		HomeserverName:      "my.server",
	}}
	bridge.OnSlackMessage(slack.Message{
		Type:    "message",
		Channel: "BOWLINGALLEY",
		TS:      "10",
		User:    slackUser,
		Text:    message,
	})
	select {
	case _ = <-called:
		return
	case _ = <-time.After(50 * time.Millisecond):
		t.Fatalf("Didn't get expected call")
	}
}

func assertUrlValueEquals(t *testing.T, v url.Values, key, want string) {
	if got := v.Get(key); got != want {
		t.Errorf("%s: want: %q got %q", key, want, got)
	}
}

func TestLoadsConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	file := path.Join(dir, "sqlite3.db")
	db := makeDBAt(t, file)
	bridge := makeBridge(t, db)

	slackMessage := &slack.Message{
		Type:    "message",
		Channel: "CANTINA",
		User:    "U34",
		Text:    "Take more chances",
		TS:      "10",
	}
	if !bridge.RoomMap.ShouldNotify(slackMessage) {
		t.Errorf("want should notify, got should not notify")
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Error closing db: %v", err)
	}

	db, err = sql.Open("sqlite3", file)
	if err != nil {
		t.Fatal(err)
	}
	bridge = makeBridge(t, db)
	if bridge.RoomMap.ShouldNotify(slackMessage) {
		t.Errorf("want should not notify, got should notify")
	}
}

func makeBridge(t *testing.T, db *sql.DB) *Bridge {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}

	echoSuppresser := matrix.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := &matrix.User{"@nancy:st.andrews", mockMatrixClient}
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	// Subsequent calls should load link from database, but don't yet
	if err := rooms.Link("!abc123:matrix.org", "CANTINA"); err != nil {
		t.Fatalf("Error linking rooms: %v", err)
	}

	return &Bridge{users, rooms, nil, http.Client{}, echoSuppresser, Config{}}
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

func (m *MockMatrixClient) AccessToken() string {
	return ""
}

func (m *MockMatrixClient) Homeserver() string {
	return ""
}

type MockSlackClient struct {
	calls []call
}

func (m *MockSlackClient) SendText(channelID, text string) error {
	m.calls = append(m.calls, call{"SendText", []interface{}{channelID, text}})
	return nil
}

func (m *MockSlackClient) AccessToken() string {
	return "slack_access_token"
}

type spyRoundTripper struct {
	fn func(*http.Request) string
}

func (r *spyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := r.fn(req)
	if resp == "" {
		resp = `{"ok": true}`
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(resp)),
	}, nil
}
