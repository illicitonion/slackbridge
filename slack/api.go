package slack

type Client interface {
	SendText(channelID, text string) error
}
