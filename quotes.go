package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Quotes struct {
	mu           sync.RWMutex
	source       string
	quotes       []*Quote
	authorMap    map[string][]*Quote
	recentQuotes map[*Quote]bool
}

func NewQuotes(ctx context.Context, source string) (*Quotes, error) {
	quotes := &Quotes{
		source:       source,
		authorMap:    map[string][]*Quote{},
		recentQuotes: map[*Quote]bool{},
	}

	if err := quotes.update(); err != nil {
		return nil, err
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		for {
			select {
			case <-ticker.C:
				if err := quotes.update(); err != nil {
					handleError(err, false)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	slog.Info("Created new quote store", slog.String("source", source))
	return quotes, nil
}

func (q *Quotes) update() error {
	var quotes []string
	var err error
	switch q.source {
	case "GITHUB":
		quotes, err = getQuotesFromGithub()
	default:
		err = errors.New(fmt.Sprintf(`Unknown quote source "%s"`, q.source))
	}

	if err != nil {
		return errors.Wrapf(err, "Failed to update quotes from %s", q.source)
	}

	q.clearQuotes()
	for _, quote := range quotes {
		q.loadQuote(quote)
	}

	slog.Info("Updated quotes", slog.Int("count", len(quotes)))

	return nil
}

func (q *Quotes) loadQuote(quoteString string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	quote := NewQuote(len(q.quotes)+1, quoteString)
	q.quotes = append(q.quotes, quote)

	for _, author := range quote.getAuthors() {
		q.authorMap[author] = append(q.authorMap[author], quote)
	}
}

func (q *Quotes) clearQuotes() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.quotes = nil
	clear(q.authorMap)
	clear(q.recentQuotes)
}

func (q *Quotes) getRandomQuote() *Quote {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.quotes) == 0 {
		return nil
	}

	candidates := make([]*Quote, len(q.quotes))
	copy(candidates, q.quotes)

	return q.pickRandomQuote(candidates)
}

func (q *Quotes) getQuoteAbout(query string) *Quote {
	var candidates = q.getAllQuotesAbout(query)

	if len(candidates) == 0 {
		return nil
	}

	return q.pickRandomQuote(candidates)
}

func (q *Quotes) getAllQuotesAbout(query string) []*Quote {
	q.mu.RLock()
	defer q.mu.RUnlock()

	query = strings.ToLower(query)

	var candidates []*Quote
	for _, quote := range q.quotes {
		if quote.matchContent(query) {
			candidates = append(candidates, quote)
		}
	}

	return candidates
}

func (q *Quotes) getQuoteBy(query string) *Quote {
	candidates := q.getAllQuotesBy(query)

	if len(candidates) == 0 {
		return nil
	}

	return q.pickRandomQuote(candidates)
}

func (q *Quotes) getAllQuotesBy(query string) []*Quote {
	q.mu.RLock()
	defer q.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	indexQuery := toIndexString(query)

	// 1. Exact match of query
	if quotes, match := q.authorMap[query]; match {
		return quotes
	}

	// 2. Exact match of index string
	if quotes, match := q.authorMap[indexQuery]; match {
		return quotes
	}

	var candidates []*Quote

	// 3. Partial match from start
	for author, quotes := range q.authorMap {
		if strings.HasPrefix(author, indexQuery) {
			candidates = append(candidates, quotes...)
		}
	}
	if len(candidates) > 0 {
		return candidates
	}

	// 4. Partial match from centre
	for author, quotes := range q.authorMap {
		if strings.Contains(author, indexQuery) {
			candidates = append(candidates, quotes...)
		}
	}

	return candidates
}

func (q *Quotes) pickRandomQuote(candidates []*Quote) *Quote {
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

	// Check whether the quote has recently been selected
	for _, quote := range candidates {
		if !q.recentQuotes[quote] {
			q.recentQuotes[quote] = true
			return quote
		}
	}

	// All quotes have recently been used, reset
	clear(q.recentQuotes)
	quote := candidates[0]
	q.recentQuotes[quote] = true
	return quote
}
