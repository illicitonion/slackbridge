package bridge

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matrix-org/slackbridge/common"
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

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := matrix.NewUser("@nancy:st.andrews", mockMatrixClient)
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, nil, http.Client{}, echoSuppresser, Config{}}
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

func TestSlackMeMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "CANTINA")

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := matrix.NewUser("@nancy:st.andrews", mockMatrixClient)
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, nil, http.Client{}, echoSuppresser, Config{}}
	bridge.OnSlackMessage(slack.Message{
		Type:    "message",
		Channel: "CANTINA",
		User:    "U34",
		Subtype: "me_message",
		Text:    "takes more chances",
	})

	want := []call{call{"SendEmote", []interface{}{"!abc123:matrix.org", "takes more chances"}}}
	if !reflect.DeepEqual(mockMatrixClient.calls, want) {
		t.Fatalf("Wrong Matrix calls, want %v got %v", want, mockMatrixClient.calls)
	}
}

func TestSlackMessageWithImage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "CANTINA")

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := matrix.NewUser("@nancy:st.andrews", mockMatrixClient)
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, nil, http.Client{}, echoSuppresser, Config{}}

	imageURL := "https://slack-files.com/files-pub/T02TMLW97-F0D2M81QA-38528eaf47/otters.jpg"
	bridge.OnSlackMessage(slack.Message{
		Type:    "message",
		Channel: "CANTINA",
		User:    "U34",
		Text:    "Cute otter",
		File: &slack.File{
			MIMEType:       "image/jpeg",
			URL:            imageURL,
			OriginalHeight: 768,
			OriginalWidth:  1024,
			Size:           90,
			CommentsCount:  1,
			InitialComment: &slack.Comment{
				Comment: "omg",
				User:    "U34",
			},
		},
	})

	want := []call{
		call{"SendImage", []interface{}{"!abc123:matrix.org", "otters.jpg", matrix.Image{
			URL: imageURL,
			Info: &matrix.ImageInfo{
				Width:    1024,
				Height:   768,
				MIMEType: "image/jpeg",
				Size:     90,
			},
		}}},
		call{"SendText", []interface{}{"!abc123:matrix.org", "omg"}},
	}
	if !reflect.DeepEqual(mockMatrixClient.calls, want) {
		t.Fatalf("Wrong Matrix calls, want:\n%v\ngot:\n%v", want, mockMatrixClient.calls)
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

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := matrix.NewUser("@sean:st.andrews", mockMatrixClient)
	slackUser := &slack.User{"U35", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, nil, http.Client{}, echoSuppresser, Config{}}
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

func TestMatrixImageMessage(t *testing.T) {
	mockMatrixClient := &MockMatrixClient{}
	mockSlackClient := &MockSlackClient{}

	db := makeDB(t)
	rooms, err := NewRoomMap(db)
	if err != nil {
		t.Fatal(err)
	}
	rooms.Link("!abc123:matrix.org", "BOWLINGALLEY")

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := matrix.NewUser("@sean:st.andrews", mockMatrixClient)
	slackUser := &slack.User{"U35", mockSlackClient}
	users.Link(matrixUser, slackUser)

	bridge := Bridge{users, rooms, nil, nil, http.Client{}, echoSuppresser, Config{
		HomeserverBaseURL: "https://some.url:1234",
	}}
	bridge.OnMatrixRoomMessage(matrix.RoomMessage{
		Type:    "m.room.message",
		Content: []byte(`{"msgtype": "m.image", "body": "It's Nancy!", "url": "mxc://some.homeserver/abcDEF"}`),
		UserID:  "@sean:st.andrews",
		RoomID:  "!abc123:matrix.org",
	})

	want := []call{call{"SendImage", []interface{}{"BOWLINGALLEY", "It's Nancy!", "https://some.url:1234/_matrix/media/v1/download/some.homeserver/abcDEF"}}}
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

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}

	slackRoomMembers := slack.NewRoomMembers()
	slackRoomMembers.Add(slackChannel, &slack.User{"someone", &MockSlackClient{}})

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
	bridge := Bridge{users, rooms, slackRoomMembers, nil, client, echoSuppresser, Config{}}
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

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}

	slackRoomMembers := slack.NewRoomMembers()
	slackRoomMembers.Add(slackChannel, &slack.User{"someone", &MockSlackClient{}})

	calledTwice := make(chan struct{}, 1)
	var invites int32
	var joins int32
	var calls int32
	verify := func(req *http.Request) string {
		if req.URL.Path == "/api/users.info" {
			return `{"ok": true, "user": {"id": "` + slackUser + `", "name": "someoneonslack"}}`
		}

		if req.URL.Path == "/_matrix/client/api/v1/rooms/"+matrixRoom+"/join" {
			atomic.AddInt32(&joins, 1)
			return ""
		}
		if req.URL.Path == "/_matrix/client/api/v1/rooms/"+matrixRoom+"/invite" {
			atomic.AddInt32(&invites, 1)
			return ""
		}
		if req.URL.Path != "/_matrix/client/api/v1/rooms/"+matrixRoom+"/send/m.room.message" {
			t.Fatalf("Got request to unexpected path %q", req.URL.Path)
			return ""
		}

		if atomic.LoadInt32(&invites) == 0 {
			t.Errorf("Didn't get expected invite before message send")
		}
		if atomic.LoadInt32(&joins) == 0 {
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
		callsAfter := atomic.AddInt32(&calls, 1)
		if callsAfter == 2 {
			calledTwice <- struct{}{}
		}
		return ""
	}
	client := http.Client{
		Transport: &spyRoundTripper{verify},
	}
	matrixUsers := matrix.NewUsers()
	bridge := Bridge{users, rooms, slackRoomMembers, matrixUsers, client, echoSuppresser, Config{
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
	bridge.OnSlackMessage(slack.Message{
		Type:    "message",
		Channel: "BOWLINGALLEY",
		TS:      "11",
		User:    slackUser,
		Text:    message,
	})
	select {
	case _ = <-calledTwice:
		if got := atomic.LoadInt32(&joins); got != 1 {
			t.Errorf("join count: want: %d, got: %d", 1, got)
		}
		return
	case _ = <-time.After(50 * time.Millisecond):
		t.Fatalf("Didn't get expected calls")
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

	echoSuppresser := common.NewEchoSuppresser()
	users, err := NewUserMap(db, http.Client{}, rooms, echoSuppresser)
	if err != nil {
		t.Fatal(err)
	}
	matrixUser := matrix.NewUser("@nancy:st.andrews", mockMatrixClient)
	slackUser := &slack.User{"U34", mockSlackClient}
	users.Link(matrixUser, slackUser)

	// Subsequent calls should load link from database, but don't yet
	if err := rooms.Link("!abc123:matrix.org", "CANTINA"); err != nil {
		t.Fatalf("Error linking rooms: %v", err)
	}

	return &Bridge{users, rooms, nil, nil, http.Client{}, echoSuppresser, Config{}}
}
