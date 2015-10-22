package matrix

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

const pathPrefix = "/_matrix/client/api/v1"

type MessageFilter func(string) bool

func AlwaysNotify(string) bool {
	return true
}

// urlBase: http(s)://host(:port)
func NewClient(accessToken string, httpClient http.Client, urlBase string, echoSuppresser *EchoSuppresser) *client {
	return &client{
		accessToken:    accessToken,
		asUser:         "",
		client:         httpClient,
		urlBase:        urlBase,
		echoSuppresser: echoSuppresser,
	}
}

func NewBotClient(accessToken, userID string, httpClient http.Client, urlBase string, echoSuppresser *EchoSuppresser) *client {
	return &client{
		accessToken:    accessToken,
		asUser:         userID,
		client:         httpClient,
		urlBase:        urlBase,
		echoSuppresser: echoSuppresser,
	}
}

type client struct {
	accessToken    string
	asUser         string
	client         http.Client
	urlBase        string
	echoSuppresser *EchoSuppresser

	mu                  sync.Mutex
	roomMessageHandlers []func(RoomMessage)
}

func (c *client) Homeserver() string {
	return c.urlBase
}

func (c *client) AccessToken() string {
	return c.accessToken
}

func (c *client) Listen(cancel chan struct{}) {
	ch := make(chan *http.Response)
	var last string
	for {
		qs := c.querystring()
		if last != "" {
			qs += "&from=" + last
		}
		req, err := http.NewRequest("GET", c.urlBase+pathPrefix+"/events"+qs, nil)
		if err != nil {
			log.Printf("Error making HTTP request: %v", err)
			continue
		}
		go c.poll(ch, req)
		select {
		case resp := <-ch:
			if resp == nil {
				continue
			}
			last = c.parseResponse(resp.Body)
		case <-cancel:
			if transport, ok := (c.client.Transport).(*http.Transport); ok {
				transport.CancelRequest(req)
				return
			}
		}
	}
}

func (c *client) parseResponse(body io.ReadCloser) string {
	defer body.Close()
	var er eventsReply
	dec := json.NewDecoder(body)
	if err := dec.Decode(&er); err != nil {
		log.Printf("Error decoding json: %v", err)
		return ""
	}
	for _, raw := range er.Chunk {
		log.Printf("Got matrix event: %s", string(raw))
		var t typedThing
		if err := json.Unmarshal(raw, &t); err != nil {
			log.Printf("Error finding type: %v", err)
			continue
		}
		switch t.Type {
		case "m.room.message":
			log.Println("Got m.room.message")
			var roomMessage RoomMessage
			if err := json.Unmarshal(raw, &roomMessage); err != nil {
				log.Printf("Error decoding inner json: %v", err)
				continue
			}
			if c.echoSuppresser.WasSent(roomMessage.EventID) {
				log.Printf("Skipping filtered message: %v", roomMessage)
				continue
			}
			if len(c.roomMessageHandlers) == 0 {
				log.Printf("No listeners for room message events")
			}
			for _, h := range c.roomMessageHandlers {
				log.Printf("Sending received matrix message to handler")
				h(roomMessage)
			}
		default:
			log.Printf("Ignoring unknown event %q", string(raw))
		}
	}
	return er.End
}

func (c *client) poll(ch chan *http.Response, req *http.Request) {
	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Error from HTTP request: %v", err)
	}
	ch <- resp
}

type eventsReply struct {
	Chunk []json.RawMessage `json:"chunk"`
	End   string            `json:"end"`
}

type typedThing struct {
	Type string `json:"type"`
}

func (c *client) OnRoomMessage(h func(RoomMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roomMessageHandlers = append(c.roomMessageHandlers, h)
}

func (c *client) SendText(roomID, text string) error {
	message := &TextMessageContent{
		Body:    text,
		MsgType: "m.text",
	}

	r, w := io.Pipe()
	go func() {
		enc := json.NewEncoder(w)
		enc.Encode(message)
		w.Close()
	}()

	url := c.urlBase + pathPrefix + "/rooms/" + roomID + "/send/m.room.message" + c.querystring()
	resp, err := c.client.Post(url, "application/json", r)
	if err != nil {
		return fmt.Errorf("error from homeserver: %v", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from homeserver: %v", err)
	}
	var e eventSendResponse
	if err := json.Unmarshal(b, &e); err != nil {
		log.Printf("Error unmarshaling event send response: %v (%s)", err, string(b))
	} else {
		c.echoSuppresser.Sent(e.EventID)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error from homeserver: %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *client) JoinRoom(roomID string) error {
	url := c.urlBase + pathPrefix + "/rooms/" + roomID + "/join" + c.querystring()
	resp, err := c.client.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		return fmt.Errorf("error from homeserver: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response from homeserver: %v", err)
		}
		return fmt.Errorf("error from homeserver: %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *client) ListRooms() (map[string]bool, error) {
	url := c.urlBase + pathPrefix + "/initialSync" + c.querystring() + "&limit=1"
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error from homeserver: %v", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from homeserver: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error from homeserver: %d: %s", resp.StatusCode, string(b))
	}
	var r initialSyncResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("error unmarshaling initialSync response: %v", err)
	}
	rooms := make(map[string]bool)
	for _, room := range r.Rooms {
		if room.Membership == "join" {
			rooms[room.RoomID] = true
		}
	}
	return rooms, nil
}

type initialSyncResponse struct {
	Rooms []initialSyncResponseRoom `json:"rooms"`
}

type initialSyncResponseRoom struct {
	Membership string `json:"membership"`
	RoomID     string `json:"room_id"`
}

func (c *client) querystring() string {
	qs := "?access_token=" + c.accessToken
	if c.asUser != "" {
		qs += "&user_id=" + c.asUser
	}
	return qs
}

type eventSendResponse struct {
	EventID string `json:"event_id"`
}

func NewEchoSuppresser() *EchoSuppresser {
	return &EchoSuppresser{
		sentEvents: make(map[string]bool),
	}
}

type EchoSuppresser struct {
	sentEvents map[string]bool // eventID -> didSend
	mu         sync.RWMutex
}

func (s *EchoSuppresser) Sent(eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentEvents[eventID] = true
}

func (s *EchoSuppresser) WasSent(eventID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sentEvents[eventID]
}
