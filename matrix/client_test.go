package matrix

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matrix-org/slackbridge/common"
)

func TestSendTextMessage(t *testing.T) {
	var called int32
	s := httptest.NewServer(&handler{t, &called, func(req *http.Request) bool {
		// I don't know why Go chooses to escape the ! but not the : even though url.QueryEscape escapes both of them
		if req.URL.String() != "/_matrix/client/api/v1/rooms/%21undertheclock:waterloo.station/send/m.room.message?access_token=6000000000peopleandyou" {
			return false
		}
		dec := json.NewDecoder(req.Body)
		var c TextMessageContent
		if err := dec.Decode(&c); err != nil {
			log.Printf("Error decoding json: %v", err)
			return false
		}
		return c.Body == "quid pro quo" && c.MsgType == "m.text"
	}})
	defer s.Close()
	c := NewClient("6000000000peopleandyou", http.Client{}, s.URL, common.NewEchoSuppresser())
	c.SendText("!undertheclock:waterloo.station", "quid pro quo")
	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("Didn't get expected HTTP request, got: %d", got)
	}
}

func TestSendEmoteMessage(t *testing.T) {
	var called int32
	s := httptest.NewServer(&handler{t, &called, func(req *http.Request) bool {
		// I don't know why Go chooses to escape the ! but not the : even though url.QueryEscape escapes both of them
		if req.URL.String() != "/_matrix/client/api/v1/rooms/%21undertheclock:waterloo.station/send/m.room.message?access_token=6000000000peopleandyou" {
			return false
		}
		dec := json.NewDecoder(req.Body)
		var c TextMessageContent
		if err := dec.Decode(&c); err != nil {
			log.Printf("Error decoding json: %v", err)
			return false
		}
		// Should probaby test the body too...
		return c.Body == "puts the fire out" && c.MsgType == "m.emote"
	}})
	defer s.Close()
	c := NewClient("6000000000peopleandyou", http.Client{}, s.URL, common.NewEchoSuppresser())
	c.SendEmote("!undertheclock:waterloo.station", "puts the fire out")
	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("Didn't get expected HTTP request, got: %d", got)
	}
}

func TestListenOneRoomMessage(t *testing.T) {
	listenTest(t, common.NewEchoSuppresser(), func(called chan struct{}) {
		select {
		case _ = <-called:
			return
		case _ = <-time.After(50 * time.Millisecond):
			t.Fatalf("Timed out waiting for event")
		}
	})
}

func TestSuppressEcho(t *testing.T) {
	echoSuppresser := common.NewEchoSuppresser()
	echoSuppresser.Sent("abc123:some.server")
	listenTest(t, echoSuppresser, func(called chan struct{}) {
		select {
		case _ = <-called:
			t.Fatalf("Should not have been called")
		case _ = <-time.After(50 * time.Millisecond):
			return
		}
	})
}

func listenTest(t *testing.T, echoSuppresser *common.EchoSuppresser, verify func(chan struct{})) {
	s := httptest.NewServer(&stubHandler{`{
	"chunk": [{
	  "content": {
	    "body": "I'm a firewoman",
	    "msgtype": "m.text"
	  },
	  "room_id": "!cantina:london",
	  "type": "m.room.message",
	  "user_id": "@nancy:london",
	  "event_id": "abc123:some.server"
	}],
	"start": "1",
	"end": "1"
}`})
	defer s.Close()

	called := make(chan struct{}, 1)

	c := NewClient("6000000000peopleandyou", http.Client{}, s.URL, echoSuppresser)

	c.OnRoomMessage(func(m RoomMessage) {
		if m.RoomID != "!cantina:london" {
			t.Errorf("RoomID: want %q got %q", "!cantina:london", m.RoomID)
		}
		if m.UserID != "@nancy:london" {
			t.Errorf("UserID: want %q got %q", "@nancy:london", m.UserID)
		}
		called <- struct{}{}
	})
	ch := make(chan struct{}, 1)
	defer func() { ch <- struct{}{} }()
	go c.Listen(ch)
	verify(called)
}

func TestMemberEvent(t *testing.T) {
	s := httptest.NewServer(&stubHandler{`{
	"chunk": [{
	  "content": {
	    "membership": "join",
	    "displayname": "ME!"
	  },
	  "room_id": "!cantina:london",
	  "type": "m.room.member",
	  "state_key": "@nancy:london",
	  "user_id": "@nancy:london",
	  "event_id": "abc123:some.server"
	}],
	"start": "1",
	"end": "1"
}`})
	defer s.Close()

	called := make(chan struct{}, 1)

	c := NewClient("6000000000peopleandyou", http.Client{}, s.URL, common.NewEchoSuppresser())

	c.OnRoomMember(func(m RoomMemberEvent) {
		if m.RoomID != "!cantina:london" {
			t.Errorf("RoomID: want %q got %q", "!cantina:london", m.RoomID)
		}
		if m.UserID != "@nancy:london" {
			t.Errorf("UserID: want %q got %q", "@nancy:london", m.UserID)
		}
		if m.StateKey != "@nancy:london" {
			t.Errorf("StateKey: want %q got %q", "@nancy:london", m.StateKey)
		}
		if m.Content.Membership != "join" {
			t.Errorf("Membership: want %q got %q", "join", m.Content.Membership)
		}
		if m.Content.DisplayName != "ME!" {
			t.Errorf("DisplayName: want %q got %q", "ME!", m.Content.DisplayName)
		}
		called <- struct{}{}
	})
	ch := make(chan struct{}, 1)
	defer func() { ch <- struct{}{} }()
	go c.Listen(ch)

	select {
	case _ = <-called:
		return
	case _ = <-time.After(50 * time.Millisecond):
		t.Fatalf("Timed out waiting for event")
	}
}

type handler struct {
	t      *testing.T
	called *int32
	filter func(*http.Request) bool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !h.filter(req) {
		h.t.Errorf("Unexpected HTTP %s to: %s", req.Method, req.URL)
	}
	atomic.AddInt32(h.called, 1)
	io.WriteString(w, "{}")
}

type stubHandler struct {
	response string
}

func (h *stubHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, h.response)
}
