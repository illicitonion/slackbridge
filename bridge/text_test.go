package bridge

import "testing"

func TestMatrixToSlack_TestEscapesSpecialCharacters(t *testing.T) {
	matrix := "<special & characters>"
	slack := "&lt;special &amp; characters&gt;"
	if got := matrixToSlack(matrix); got != slack {
		t.Errorf("matrixToSlack(%s): want %q got %q", matrix, slack, got)
	}
}

func TestSlackToMatrix_TestUnescapesSpecialCharacters(t *testing.T) {
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

func testSlackToMatrix(t *testing.T, slack, matrix string) {
	if got := slackToMatrix(slack); got != matrix {
		t.Errorf("slackToMatrix(%s): want %q got %q", slack, matrix, got)
	}
}
