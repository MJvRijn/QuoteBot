package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Quote struct {
	idx              int
	raw              string
	author           string
	authorLowercase  string
	content          string
	contentLowercase string
}

func NewQuote(idx int, raw string) *Quote {
	quote := Quote{idx: idx, raw: raw}

	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		quote.author = strings.TrimSpace(parts[0])
		quote.content = strings.TrimSpace(parts[1])
	} else {
		quote.content = strings.TrimSpace(parts[0])
	}

	quote.authorLowercase = strings.ToLower(quote.author)
	quote.contentLowercase = strings.ToLower(quote.content)

	return &quote
}

func (q *Quote) matchContent(query string) bool {
	return strings.Contains(q.contentLowercase, query)
}

func (q *Quote) getAuthors() []string {
	// Replicate derfymatch from sourcemod plugin, thanks I hate it too :)

	authors := []string{q.authorLowercase}
	indexAuthor := toIndexString(q.author)

	lowerIndexAuthor := strings.ToLower(indexAuthor)
	if lowerIndexAuthor != q.authorLowercase {
		authors = append(authors, lowerIndexAuthor)
	}

	fullAuthor := append([]rune(indexAuthor), 0)         // Pretend it's a c-string so I don't have to change the loop :D
	for i, lastSplit := 1, 0; i < len(fullAuthor); i++ { // Start at second rune
		r := fullAuthor[i]
		finalRune := r == 0
		previousIsLowercase := unicode.IsLower(fullAuthor[i-1])
		nextIsLetter := !finalRune && unicode.IsLetter(fullAuthor[i+1])
		caseSplit := unicode.IsUpper(r) && previousIsLowercase && nextIsLetter

		if unicode.IsSpace(r) || caseSplit || finalRune {
			shortAuthor := strings.TrimSpace(strings.ToLower(string(fullAuthor[lastSplit:i])))
			if shortAuthor != lowerIndexAuthor { // Don't add duplicate of full author
				authors = append(authors, shortAuthor)
			}
			lastSplit = i
		}
	}

	return authors
}

func (q *Quote) toString() string {
	if q == nil {
		return "Nil Quote"
	}

	if q.author == "" {
		return q.content
	}

	return fmt.Sprintf("(#%04d) %s: %s", q.idx, q.author, q.content)
}

func (q *Quote) toDiscordString() string {
	if q.author == "" {
		return fmt.Sprintf(">>> %s", q.content)
	}

	return fmt.Sprintf(">>> **%s**:\n%s", q.author, q.content)
}
