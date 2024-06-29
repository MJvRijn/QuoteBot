package main

import (
	"fmt"
	"strings"
)

type Quote struct {
	raw              string
	author           string
	authorLowercase  string
	content          string
	contentLowercase string
}

func NewQuote(raw string) *Quote {
	quote := Quote{raw: raw}

	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		quote.author = parts[0]
		quote.content = parts[1]
	} else {
		quote.content = parts[0]
	}

	quote.authorLowercase = strings.ToLower(quote.author)
	quote.contentLowercase = strings.ToLower(quote.content)

	return &quote
}

func (q *Quote) matchContent(query string) bool {
	return strings.Contains(q.contentLowercase, query)
}

func (q *Quote) matchAuthor(query string) bool {
	return strings.Contains(q.authorLowercase, query)
}

func (q *Quote) discordFormat() string {
	return fmt.Sprintf(">>> **%s**:\n%s", q.author, q.content)
}
