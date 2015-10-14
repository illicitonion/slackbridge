package matrix

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

const pathPrefix = "/_matrix/client/api/v1"

// urlBase: http(s)://host(:port)
func NewClient(accessToken string, httpClient http.Client, urlBase string) *client {
	return &client{
		accessToken: accessToken,
		client:      httpClient,
		urlBase:     urlBase,
	}
}

type client struct {
	accessToken string
	client      http.Client
	urlBase     string

	mu                  sync.Mutex
	roomMessageHandlers []func(RoomMessage)
}

func (c *client) Listen(cancel chan struct{}) {
	ch := make(chan *http.Response)
	for {
		req, err := http.NewRequest("GET", c.urlBase+pathPrefix+"/events?access_token="+c.accessToken, nil)
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
			c.parseResponse(resp.Body)
		case <-cancel:
			if transport, ok := (c.client.Transport).(*http.Transport); ok {
				transport.CancelRequest(req)
				return
			}
		}
	}
}

func (c *client) parseResponse(body io.ReadCloser) {
	defer body.Close()
	var er eventsReply
	dec := json.NewDecoder(body)
	if err := dec.Decode(&er); err != nil {
		log.Printf("Error decoding json: %v", err)
		return
	}
	for _, raw := range er.Chunk {
		var t typedThing
		if err := json.Unmarshal(raw, &t); err != nil {
			log.Printf("Error finding type: %v", err)
			return
		}
		switch t.Type {
		case "m.room.message":
			var roomMessage RoomMessage
			if err := json.Unmarshal(raw, &roomMessage); err != nil {
				log.Printf("Error decoding inner json: %v", err)
				return
			}
			if len(c.roomMessageHandlers) == 0 {
				log.Printf("No listeners for room message events")
			}
			for _, h := range c.roomMessageHandlers {
				h(roomMessage)
			}
		default:
			log.Printf("Ignoring unknown event %q", string(raw))
		}
	}
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

	url := c.urlBase + pathPrefix + "/rooms/" + roomID + "/send/m.room.message?access_token=" + c.accessToken
	resp, err := c.client.Post(url, "application/json", r)
	if err != nil {
		return fmt.Errorf("error from homeserver: %v", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from homeserver: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error from homeserver: %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
