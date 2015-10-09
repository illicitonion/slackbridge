package matrix

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const pathPrefix = "/_matrix/client/api/v1"

// urlBase: http(s)://host(:port)
func NewClient(userID, accessToken string, httpClient http.Client, urlBase string) *client {
	return &client{
		userID:      userID,
		accessToken: accessToken,
		client:      httpClient,
		urlBase:     urlBase,
	}
}

type client struct {
	userID      string
	accessToken string
	client      http.Client
	urlBase     string
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
