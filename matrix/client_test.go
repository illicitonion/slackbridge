package matrix

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
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
	c := NewClient("@jack:waterloo.station", "6000000000peopleandyou", http.Client{}, s.URL)
	c.SendText("!undertheclock:waterloo.station", "quid pro quo")
	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("Didn't get expected HTTP request, got: %d", got)
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
