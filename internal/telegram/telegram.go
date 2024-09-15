// Package telegram encapsulates transport layer (not OSI, but generally telegram receive/send handlers).
package telegram

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"ole-bot/internal/dispatcher"
)

// Deps is a carrier of options for Bot.
type Deps struct {
	EventDispatcher *dispatcher.EventDispatcher
}

// Opts is a carrier of options for Bot.
type Opts struct {
	Token string
	Debug bool
}

// Bot wraps a third-party telegram API implementation.
type Bot struct {
	api             *tgbotapi.BotAPI
	eventDispatcher *dispatcher.EventDispatcher
}

// NewBot instantiates underlying BotAPI instance and returns a new configured Bot.
func NewBot(deps Deps, opts Opts) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(opts.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to init bot-api instance: %s", err)
	}
	slog.Info("Authorized on account", "name", api.Self.UserName)
	api.Debug = opts.Debug

	return &Bot{
		api:             api,
		eventDispatcher: deps.EventDispatcher,
	}, nil
}

// Listen runs update receiving loop. Correct messages are provided to dispatcher.
func (b *Bot) Listen(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return
		case update := <-updates:
			// TODO: This check seems to be redundant, but it was provided in a repository example.
			// 	Maybe I'll check if this is possible later.
			if update.Message == nil {
				slog.WarnContext(ctx, "got empty message in update", "update_id", update.UpdateID)
				continue
			}
			response := b.eventDispatcher.DispatchMessage(ctx, update.Message)

			slog.InfoContext(ctx, fmt.Sprintf("[%s] %s", update.Message.From.UserName, update.Message.Text))
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
			msg.ReplyToMessageID = update.Message.MessageID

			b.api.Send(msg)
		}
	}
}
