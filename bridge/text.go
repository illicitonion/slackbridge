package bridge

import (
	"regexp"
	"strings"
)

func matrixToSlack(matrix string) string {
	m := matrix
	m = strings.Replace(m, "&", "&amp;", -1)
	m = strings.Replace(m, ">", "&gt;", -1)
	m = strings.Replace(m, "<", "&lt;", -1)
	return m
}

func slackToMatrix(slack string) string {
	s := slack
	s = strings.Replace(s, "&lt;", "<", -1)
	s = strings.Replace(s, "&gt;", ">", -1)
	s = strings.Replace(s, "&amp;", "&", -1)

	if matched, _ := regexp.MatchString(`:[^:\s]*:`, s); matched {
		s = replaceEmoji(s)
	}

	return s
}

func replaceEmoji(s string) string {
	for text, emojum := range emoji {
		s = strings.Replace(s, text, emojum, -1)
	}
	return s
}
