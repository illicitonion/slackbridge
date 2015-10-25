package slack

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func WriteStuff(ws *websocket.Conn) {
	io.WriteString(ws, "Hello")
}

func TestReceiveHello(t *testing.T) {
	want := Hello{Type: "hello"}
	do := func(client *client, called func()) {
		client.OnHello(func(h Hello) {
			if h.Type != "hello" {
				t.Errorf("want %q got %q", "hello", h.Type)
			}
			called()
		})
	}
	testReceive(t, want, do, AlwaysNotify)
}

func TestReceiveMessage(t *testing.T) {
	want := Message{
		Type: "message",
		User: "nancy",
		Text: "I'm a... firewoman",
	}
	do := func(client *client, called func()) {
		client.OnMessage(func(got Message) {
			if want != got {
				t.Errorf("want %v got %v", want, got)
			}
			called()
		})
	}
	testReceive(t, want, do, AlwaysNotify)
}

func TestIgnoresFilteredMessages(t *testing.T) {
	want := Message{
		Type: "message",
		User: "nancy",
		Text: "I'm a... firewoman",
	}

	client, cancel, closeFn := stubEvent(t, want, func(m *Message) bool { return false })
	defer closeFn()

	called := make(chan struct{})

	client.OnMessage(func(got Message) {
		called <- struct{}{}
	})

	go func() {
		if err := client.Listen(cancel); err != nil {
			t.Fatalf("Error listening: %v", err)
		}
	}()

	select {
	case _ = <-called:
		t.Fatalf("Got unexpected message")
	case _ = <-time.After(50 * time.Millisecond):
		return
	}
}

func TestSendMessage(t *testing.T) {
	text := "It's a grand gesture"
	do := func(client Client) error {
		return client.SendText("CANTINA", text)
	}
	verify := func(v url.Values) bool {
		return v.Get("text") == text
	}
	testSendMessage(t, do, verify)
}

func TestSendImage(t *testing.T) {
	text := "It's a grand gesture"
	imageURL := "https://some.url/image.jpg"
	do := func(client Client) error {
		return client.SendImage("CANTINA", text, imageURL)
	}
	verify := func(v url.Values) bool {
		return v.Get("fallback") == text && v.Get("image_url") == imageURL
	}
	testSendMessage(t, do, verify)
}

func testSendMessage(t *testing.T, do func(Client) error, verify func(url.Values) bool) {
	called := false
	client := NewClient("cynicism", http.Client{
		Transport: &roundTripper{
			t:        t,
			response: `{"ok": true}`,
			called:   &called,
			filter: func(req *http.Request) bool {
				if req.URL.String() != "https://slack.com/api/chat.postMessage" {
					log.Printf("Wrong URL: %q", req.URL.String())
					return false
				}
				if err := req.ParseForm(); err != nil {
					log.Printf("Error parsing form: %v", err)
					return false
				}
				if req.Form.Get("token") != "cynicism" ||
					req.Form.Get("channel") != "CANTINA" ||
					req.Form.Get("as_user") != "true" ||
					!verify(req.Form) {
					log.Printf("Got incorrect request: %v", req.Form)
					return false
				}
				return true
			},
		},
	}, AlwaysNotify)
	if err := do(client); err != nil {
		t.Errorf("Error sending message: %v", err)
	}
	if !called {
		t.Errorf("Expected HTTP request but got none or incorrect")
	}
}

func testReceive(t *testing.T, want interface{}, do func(*client, func()), filter MessageFilter) {
	client, cancel, closeFn := stubEvent(t, want, filter)
	defer closeFn()

	called := make(chan struct{})
	calledFn := func() { called <- struct{}{} }

	do(client, calledFn)

	go func() {
		if err := client.Listen(cancel); err != nil {
			t.Fatalf("Error listening: %v", err)
		}
	}()

	select {
	case _ = <-called:
		return
	case _ = <-time.After(50 * time.Millisecond):
		t.Fatalf("Timed out waiting for event")
	}
}

func stubEvent(t *testing.T, e interface{}, filter MessageFilter) (*client, chan struct{}, func()) {
	f := func(ws *websocket.Conn) {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("Error marshaling to JSON: %v", err)
		}
		ws.Write(b)
	}
	s := httptest.NewServer(websocket.Handler(f))
	rtm := rtmStartResponse{
		OK:  true,
		URL: strings.Replace(s.URL, "http://", "ws://", 1),
	}
	b, err := json.Marshal(rtm)
	if err != nil {
		t.Fatalf("Error marshaling rtmStartResponse: %v", err)
	}

	cancel := make(chan struct{}, 1)
	client := NewClient("", http.Client{
		Transport: &roundTripper{
			t:        t,
			response: string(b),
			filter: func(*http.Request) bool {
				return true
			},
		},
	}, filter)

	return client, cancel, func() {
		s.Close()
		cancel <- struct{}{}
	}
}

type stubMessageFilter struct {
	fn func(m *Message) bool
}

func (s *stubMessageFilter) ShouldNotify(m *Message) bool {
	return s.fn(m)
}

type roundTripper struct {
	t        *testing.T
	response string
	filter   func(req *http.Request) bool
	called   *bool
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !r.filter(req) {
		r.t.Errorf("Unexpected HTTP %s to: %s", req.Method, req.URL)
	}
	if r.called != nil {
		*r.called = true
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(r.response)),
	}, nil
}
