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

	s = replaceLinksAndSuch(s)

	s = strings.Replace(s, "&lt;", "<", -1)
	s = strings.Replace(s, "&gt;", ">", -1)
	s = strings.Replace(s, "&amp;", "&", -1)

	if matched, _ := regexp.MatchString(`:[^:\s]*:`, s); matched {
		s = replaceEmoji(s)
	}

	return s
}

func replaceLinksAndSuch(slack string) string {
	// TODO: <@USER>
	// TODO: <#CHANNEL>

	find := regexp.MustCompile("<[^>\x00]*>")
	for {
		match := find.FindString(slack)
		if match == "" {
			return strings.Replace(slack, "\x00", "", -1)
		}
		if len(match) <= 3 {
			continue
		}

		replacement := match[1 : len(match)-1]
		hasBang := replacement[0] == '!'
		pipe := strings.Index(replacement, "|")

		if pipe != -1 {
			link := replacement[0:pipe]
			caption := replacement[pipe+1:]
			replacement = caption
			if !hasBang && link != "" {
				replacement += " ( " + link + " )"
			}
		}

		if hasBang {
			start := 1
			if pipe != -1 {
				start = 0
			}
			replacement = "<\x00" + replacement[start:] + ">"
		}
		slack = strings.Replace(slack, match, replacement, -1)
	}
}

func replaceEmoji(s string) string {
	for text, emojum := range emoji {
		s = strings.Replace(s, text, emojum, -1)
	}
	return s
}
