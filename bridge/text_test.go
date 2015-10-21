package bridge

import "testing"

func SlackToMatrix_TestUnescapesSpecialCharacters(t *testing.T) {
	testSlackToMatrix(t, "&lt;special &amp; characters&gt;", "<special & characters>")
}

func SlackToMatrix_OneEmojum(t *testing.T) {
	testSlackToMatrix(t, ":wink:", "😉")
}

func SlackToMatrix_SeveralEmoji(t *testing.T) {
	testSlackToMatrix(t, ":wink::wink: :rugby_football:", "😉😉 🏉")
}

func SlackToMatrix_EmojiWithWhitespace(t *testing.T) {
	testSlackToMatrix(t, ":win k:", ":win k:")
}

func SlackToMatrix_ImproperlyFormedEmoji(t *testing.T) {
	testSlackToMatrix(t, ":k:wink:", ":k😉")
}

func SlackToMatrix_IgnoresUnknownEmoji(t *testing.T) {
	testSlackToMatrix(t, ":golzillavodka:", ":godzillavodka:")
}

func testSlackToMatrix(t *testing.T, slack, matrix string) {
	if got := slackToMatrix(slack); got != matrix {
		t.Errorf("slackToMatrix(%s): want %q got %q", slack, matrix, got)
	}
}
