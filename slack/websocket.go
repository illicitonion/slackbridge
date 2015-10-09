package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

const bufSize = 16 * 1024

func NewClient(c http.Client) *client {
	return &client{
		client: c,
	}
}

func (c *client) Listen(token string, cancel chan struct{}) error {
	if c.ws != nil {
		return fmt.Errorf("already listening")
	}

	url, err := c.websocketURL(token)
	if err != nil {
		return err
	}
	c.ws, err = websocket.Dial(url, "", "http://localhost")
	if err != nil {
		return fmt.Errorf("error dialing: %v", err)
	}

	ch := make(chan []byte)
	for {
		go c.read(ch)
		select {
		case b := <-ch:
			var e event
			if err := json.Unmarshal(b, &e); err != nil {
				log.Printf("Error unmarshaling websocket type: %v", err)
			}
			switch e.Type {
			case "hello":
				var h Hello
				if err := json.Unmarshal(b, &h); err != nil {
					log.Printf("Error unmarshaling websocket response: %v", err)
				}
				if len(c.helloHandlers) != 0 {
					log.Printf("No listeners for hello events")
				}
				for _, c := range c.helloHandlers {
					c(h)
				}
			case "message":
				var m Message
				if err := json.Unmarshal(b, &m); err != nil {
					log.Printf("Error unmarshaling websocket response: %v", err)
				}
				if len(c.messageHandlers) != 0 {
					log.Printf("No listeners for message events")
				}
				for _, c := range c.messageHandlers {
					c(m)
				}
			default:
				log.Printf("Ignoring unknown event: %q", string(b))
			}
		case _ = <-cancel:
			return c.ws.Close()
		}
		return nil
	}
}

func (c *client) OnHello(h func(Hello)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.helloHandlers = append(c.helloHandlers, h)
}

func (c *client) OnMessage(h func(Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageHandlers = append(c.messageHandlers, h)
}

type client struct {
	client http.Client
	ws     *websocket.Conn

	mu              sync.Mutex
	helloHandlers   []func(Hello)
	messageHandlers []func(Message)
}

func (c *client) websocketURL(token string) (string, error) {
	resp, err := c.client.Get("https://slack.com/api/rtm.start?token=" + token)
	if err != nil {
		return "", fmt.Errorf("error starting stream: %v", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading stream details: %v", err)
	}
	var r rtmStartResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}
	if !r.OK {
		log.Printf("Bad response from slack getting websocket: %v", string(b))
		return "", fmt.Errorf("bad response: %v", err)
	}
	return r.URL, nil
}

func (c *client) read(ch chan []byte) {
	b := make([]byte, bufSize)
	n, err := c.ws.Read(b)
	if err != nil {
		log.Printf("Error reading from websocket: %v", err)
	}
	ch <- b[0:n]
}

type rtmStartResponse struct {
	OK  bool   `json:"ok"`
	URL string `json:"url"`
}
