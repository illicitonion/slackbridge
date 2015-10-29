package matrix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/matrix-org/slackbridge/common"
)

const pathPrefix = "/_matrix/client/api/v1"

// urlBase: http(s)://host(:port)
func NewClient(accessToken string, httpClient http.Client, urlBase string, echoSuppresser *common.EchoSuppresser) *client {
	return &client{
		accessToken:    accessToken,
		asUser:         "",
		client:         httpClient,
		urlBase:        urlBase,
		echoSuppresser: echoSuppresser,
	}
}

func NewBotClient(accessToken, userID string, httpClient http.Client, urlBase string, echoSuppresser *common.EchoSuppresser) *client {
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
	echoSuppresser *common.EchoSuppresser

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
			c.echoSuppresser.Wait()
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

	_, err := c.postEvent(roomID, message)
	return err
}

func (c *client) SendImage(roomID, text string, image *Image) error {
	imageURL, err := c.uploadImage(image)
	if err != nil {
		return err
	}

	message := &ImageMessageContent{
		Body:    text,
		MsgType: "m.image",
		URL:     imageURL,
		Info:    image.Info,
	}

	_, err = c.postEvent(roomID, message)
	return err
}

func (c *client) SendEmote(roomID, emote string) error {
	message := &TextMessageContent{
		Body:    emote,
		MsgType: "m.emote",
	}

	_, err := c.postEvent(roomID, message)
	return err
}

func (c *client) uploadImage(image *Image) (string, error) {
	resp, err := c.client.Get(image.URL)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad response from image GET: %s", resp.Status)
	}
	defer resp.Body.Close()
	req, err := http.NewRequest("POST", c.urlBase+"/_matrix/media/v1/upload"+c.querystring(), resp.Body)
	if err != nil {
		return "", fmt.Errorf("error creating http request: %v", err)
	}

	var contentType string
	if image.Info != nil && image.Info.MIMEType != "" {
		contentType = image.Info.MIMEType
	} else {
		contentType = resp.Header.Get("Content-Type")
	}
	req.Header.Set("Content-Type", contentType)
	if image.Info != nil && image.Info.Size > 0 {
		req.ContentLength = image.Info.Size
	} else {
		length, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return "", fmt.Errorf("error parsing content-length header: %v", err)
		}
		req.ContentLength = length
	}

	uploadResp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("Bad response from image POST: %s - %s", uploadResp.Status, string(body))
	}
	dec := json.NewDecoder(uploadResp.Body)
	var ur uploadResponse
	if err := dec.Decode(&ur); err != nil {
		return "", err
	}
	return ur.ContentURI, nil
}

type uploadResponse struct {
	ContentURI string `json:"content_uri"`
}

func (c *client) postEvent(roomID string, event interface{}) (*http.Response, error) {
	r, w := io.Pipe()
	go func() {
		enc := json.NewEncoder(w)
		enc.Encode(event)
		w.Close()
	}()

	c.echoSuppresser.StartSending()
	defer c.echoSuppresser.DoneSending()

	url := c.urlBase + pathPrefix + "/rooms/" + roomID + "/send/m.room.message" + c.querystring()
	resp, err := c.client.Post(url, "application/json", r)
	if err != nil {
		return nil, fmt.Errorf("error from homeserver: %v", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("error reading response from homeserver: %v", err)
	}
	if resp.StatusCode != 200 {
		return resp, fmt.Errorf("error from homeserver: %d: %s", resp.StatusCode, string(b))
	}
	var e eventSendResponse
	if err := json.Unmarshal(b, &e); err != nil {
		log.Printf("Error unmarshaling event send response: %v (%s)", err, string(b))
	} else {
		log.Printf("Sent matrix event with ID: %s", e.EventID)
		c.echoSuppresser.Sent(e.EventID)
	}
	return resp, nil
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

func (c *client) Invite(roomID, userID string) error {
	b, err := json.Marshal(inviteBody{userID})
	if err != nil {
		return err
	}
	url := c.urlBase + pathPrefix + "/rooms/" + roomID + "/invite" + c.querystring()
	resp, err := c.client.Post(url, "application/json", bytes.NewReader(b))
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

type inviteBody struct {
	UserID string `json:"user_id"`
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
