package slack

type Client interface {
	SendText(channelID, text string) error
	SendImage(channelID, fallbackText, url string) error

	AccessToken() string
}
