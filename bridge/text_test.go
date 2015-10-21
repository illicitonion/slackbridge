package bridge

import "testing"

func TestMatrixToSlack_EscapesSpecialCharacters(t *testing.T) {
	matrix := "<special & characters>"
	slack := "&lt;special &amp; characters&gt;"
	if got := matrixToSlack(matrix); got != slack {
		t.Errorf("matrixToSlack(%s): want %q got %q", matrix, slack, got)
	}
}

func TestSlackToMatrix_UnescapesSpecialCharacters(t *testing.T) {
	testSlackToMatrix(t, "&lt;special &amp; characters&gt;", "<special & characters>")
}

func TestSlackToMatrix_OneEmojum(t *testing.T) {
	testSlackToMatrix(t, ":wink:", "😉")
}

func TestSlackToMatrix_SeveralEmoji(t *testing.T) {
	testSlackToMatrix(t, ":wink::wink: :rugby_football:", "😉😉 🏉")
}

func TestSlackToMatrix_EmojiWithWhitespace(t *testing.T) {
	testSlackToMatrix(t, ":win k:", ":win k:")
}

func TestSlackToMatrix_ImproperlyFormedEmoji(t *testing.T) {
	testSlackToMatrix(t, ":k:wink:", ":k😉")
}

func TestSlackToMatrix_IgnoresUnknownEmoji(t *testing.T) {
	testSlackToMatrix(t, ":godzillavodka:", ":godzillavodka:")
}

func TestSlackToMatrix_SimpleLink(t *testing.T) {
	testSlackToMatrix(t, "it's a <http://www.somewhere.com/path?query> link", "it's a http://www.somewhere.com/path?query link")
}

func TestSlackToMatrix_LinkWithCaption(t *testing.T) {
	testSlackToMatrix(t, "it's a <http://www.somewhere.com/path|captioned> link", "it's a captioned ( http://www.somewhere.com/path ) link")
}

func TestSlackToMatrix_UnrecognizedCommand(t *testing.T) {
	testSlackToMatrix(t, "it's <!bang>", "it's <bang>")
}

func TestSlackToMatrix_UnrecognizedCommandWithCaption(t *testing.T) {
	testSlackToMatrix(t, "it's <!bang|the dice game>", "it's <the dice game>")
}

func TestSlackToMatrix_Multiple(t *testing.T) {
	testSlackToMatrix(t, "it's <!bang> <https://www.rainbow.com>", "it's <bang> https://www.rainbow.com")
}

func testSlackToMatrix(t *testing.T, slack, matrix string) {
	if got := slackToMatrix(slack); got != matrix {
		t.Errorf("slackToMatrix(%s): want %q got %q", slack, matrix, got)
	}
}
