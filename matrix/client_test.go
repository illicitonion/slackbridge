package matrix

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendTextMessage(t *testing.T) {
	var called int32
	s := httptest.NewServer(&handler{t, &called, func(req *http.Request) bool {
		// I don't know why Go chooses to escape the ! but not the : even though url.QueryEscape escapes both of them
		if req.URL.String() != "/_matrix/client/api/v1/rooms/%21undertheclock:waterloo.station/send/m.room.message?access_token=6000000000peopleandyou" {
			return false
		}
		// Should probaby test the body too...
		return true
	}})
	defer s.Close()
	c := NewClient("6000000000peopleandyou", http.Client{}, s.URL)
	c.SendText("!undertheclock:waterloo.station", "quid pro quo")
	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("Didn't get expected HTTP request, got: %d", got)
	}
}

func TestListenOneRoomMessage(t *testing.T) {
	s := httptest.NewServer(&stubHandler{`{
	"chunk": [{
	  "content": {
	    "body": "I'm a firewoman",
	    "msgtype": "m.text"
	  },
	  "room_id": "!cantina:london",
	  "type": "m.room.message",
	  "user_id": "@nancy:london"
	}],
	"start": "1",
	"end": "1"
}`})
	defer s.Close()

	called := make(chan struct{}, 1)

	c := NewClient("6000000000peopleandyou", http.Client{}, s.URL)

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
	go c.Listen(ch)
	select {
	case _ = <-called:
		ch <- struct{}{}
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
