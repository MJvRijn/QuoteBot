package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sort"
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

	authorOption := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Required:    true,
		Name:        "name",
		Description: "Name of person who was quoted",
	}

	subjectOption := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Required:    true,
		Name:        "content",
		Description: "Content of the quote",
	}

	_, err = discord.ApplicationCommandBulkOverwrite(os.Getenv("DISCORD_APP_ID"), "", []*discordgo.ApplicationCommand{{
		Name:        "quote",
		Description: "Get quotes",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "from",
			Description: "Show a quote from a specific person",
			Options:     []*discordgo.ApplicationCommandOption{authorOption},
		}, {
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "about",
			Description: "Show a quote about a specific subject",
			Options:     []*discordgo.ApplicationCommandOption{subjectOption},
		}, {
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "listfrom",
			Description: "List all quotes from a specific person",
			Options:     []*discordgo.ApplicationCommandOption{authorOption},
		}, {
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "listabout",
			Description: "List all quotes about a specific subject",
			Options:     []*discordgo.ApplicationCommandOption{subjectOption},
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

	var selectedQuotes []*Quote
	switch subcommand.Name {
	case "about":
		if quote := quotes.getQuoteAbout(subcommand.Options[0].StringValue()); quote != nil {
			selectedQuotes = append(selectedQuotes, quote)
		}
	case "from":
		if quote := quotes.getQuoteBy(subcommand.Options[0].StringValue()); quote != nil {
			selectedQuotes = append(selectedQuotes, quote)
		}
	case "random":
		if quote := quotes.getRandomQuote(); quote != nil {
			selectedQuotes = append(selectedQuotes, quote)
		}
	case "listfrom":
		selectedQuotes = quotes.getAllQuotesBy(subcommand.Options[0].StringValue())
	case "listabout":
		selectedQuotes = quotes.getAllQuotesAbout(subcommand.Options[0].StringValue())
	}

	sort.Slice(selectedQuotes, func(i, j int) bool {
		return selectedQuotes[i].idx < selectedQuotes[j].idx
	})

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{},
	}

	var content, logQuote string
	if len(selectedQuotes) == 0 {
		content = "I wasn't able to find a matching quote"
		logQuote = "No quotes"
		response.Data.Flags |= discordgo.MessageFlagsEphemeral
	} else if len(selectedQuotes) == 1 && !strings.Contains(subcommand.Name, "list") {
		content = selectedQuotes[0].toDiscordString()
		logQuote = selectedQuotes[0].toString()
	} else {
		content = fmt.Sprintf("I found %d quote(s):\n```\n", len(selectedQuotes))
		for _, quote := range selectedQuotes {
			quoteStr := quote.toString()
			if len(content)+len(quoteStr)+1 <= 1969 {
				content += quoteStr + "\n"
			} else {
				content += "And more that don't fit...\n"
				break
			}
		}
		content += "\n```"
		logQuote = "Multiple quotes"
		response.Data.Flags |= discordgo.MessageFlagsEphemeral
	}
	response.Data.Content = content

	var userName string
	if interaction.Member != nil && interaction.Member.User != nil {
		userName = interaction.Member.User.Username
	} else if interaction.User != nil {
		userName = interaction.User.Username
	}

	slog.Info("Processed quote command",
		slog.String("subcommand", subcommand.Name),
		slog.String("user", userName),
		slog.Duration("duration", time.Since(start)),
		slog.String("quote", logQuote))

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
