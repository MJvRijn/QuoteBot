package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var Version string = "development"

func main() {
	slog.Info("Starting QuoteBot", slog.String("version", Version))

	start := time.Now()
	mainCtx := context.Background()
	quotes, err := NewQuotes(mainCtx, "GITHUB")
	if err != nil {
		handleError(err, true)
	}

	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		handleError(err, true)
	}

	_, err = discord.ApplicationCommandBulkOverwrite(os.Getenv("DISCORD_APP_ID"), "", []*discordgo.ApplicationCommand{{
		Name:        "quote",
		Description: "Show a quote",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "by",
			Description: "Show a quote by someone",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Name:        "author",
				Description: "Author of the quote",
			}},
		}, {
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "about",
			Description: "Show a quote about something",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Name:        "subject",
				Description: "Subject of the quote",
			}},
		}, {
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "random",
			Description: "Show a random quote",
		}},
	}})
	if err != nil {
		handleError(err, true)
	}

	discord.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if interaction.Type != discordgo.InteractionApplicationCommand {
			slog.Warn("Received invalid interaction")
			return
		}

		data := interaction.ApplicationCommandData()
		switch data.Name {
		case "quote":
			handleQuoteCommand(session, interaction, quotes)
		}
	})

	if err := discord.Open(); err != nil {
		handleError(err, true)
	}
	defer discord.Close()

	slog.Info("Startup complete", slog.Duration("duration", time.Since(start)))

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT)
	<-exit
	slog.Info("Shutdown signal received")
}

func handleQuoteCommand(session *discordgo.Session, interaction *discordgo.InteractionCreate, quotes *Quotes) {
	start := time.Now()

	data := interaction.ApplicationCommandData()
	subcommand := data.Options[0]

	var quote *Quote
	switch subcommand.Name {
	case "about":
		quote = quotes.getQuoteAbout(subcommand.Options[0].StringValue())
	case "by":
		quote = quotes.getQuoteBy(subcommand.Options[0].StringValue())
	case "random":
		quote = quotes.getRandomQuote()
	}

	var content = "I wasn't able to find a matching quote"
	if quote != nil {
		content = quote.toDiscordString()
	}

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	}

	if quote == nil {
		response.Data.Flags |= discordgo.MessageFlagsEphemeral
	}

	slog.Info("Processed quote command",
		slog.String("subcommand", subcommand.Name),
		slog.Duration("duration", time.Since(start)),
		slog.String("quote", quote.toString()))

	err := session.InteractionRespond(interaction.Interaction, response)
	if err != nil {
		handleError(err, false)
	}
}

func handleError(err error, fatal bool) {
	if fatal {
		panic(err)
	}
	slog.Error(err.Error())
}

var indexStringRegex = regexp.MustCompile(`[^a-zA-Z ]+`)

func toIndexString(author string) string {
	author = strings.TrimSpace(author)
	return indexStringRegex.ReplaceAllString(author, "")
}
