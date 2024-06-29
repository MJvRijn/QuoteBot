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
	mu     sync.RWMutex
	source string
	quotes []*Quote
}

func NewQuotes(ctx context.Context, source string) (*Quotes, error) {
	quotes := &Quotes{
		source: source,
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

	quote := NewQuote(quoteString)
	q.quotes = append(q.quotes, quote)
}

func (q *Quotes) clearQuotes() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.quotes = nil
}

func (q *Quotes) getRandomQuote() *Quote {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.quotes) == 0 {
		return nil
	}

	options := make([]*Quote, len(q.quotes))
	copy(options, q.quotes)

	return pickRandomQuote(options)
}

func (q *Quotes) getQuoteAbout(subject string) *Quote {
	q.mu.RLock()
	defer q.mu.RUnlock()

	subject = strings.ToLower(subject)

	var options []*Quote
	for _, quote := range q.quotes {
		if quote.matchContent(subject) {
			options = append(options, quote)
		}
	}

	if len(options) == 0 {
		return nil
	}

	return pickRandomQuote(options)
}

func (q *Quotes) getQuoteBy(author string) *Quote {
	q.mu.RLock()
	defer q.mu.RUnlock()

	author = strings.ToLower(author)

	var options []*Quote
	for _, quote := range q.quotes {
		if quote.matchAuthor(author) {
			options = append(options, quote)
		}
	}

	if len(options) == 0 {
		return nil
	}

	return pickRandomQuote(options)
}

func pickRandomQuote(options []*Quote) *Quote {
	quoteIdx := rand.Intn(len(options))
	return options[quoteIdx]
}
