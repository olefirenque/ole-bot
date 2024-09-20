package main

import (
	"context"
	"log"
	"syscall"

	"github.com/joho/godotenv"

	"ole-bot/internal/closer"
	"ole-bot/internal/dispatcher"
	"ole-bot/internal/openai"
	"ole-bot/internal/telegram"
	"ole-bot/pkg/ratelimiter"
)

func main() {
	envFile, _ := godotenv.Read(".env")
	cls := closer.NewCloser(syscall.SIGINT, syscall.SIGTERM)
	defer cls.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cls.Add(func() error {
		cancel()
		return nil
	})

	openAiClient := openai.NewClient(
		openai.Opts{
			ApiKey: envFile["OPENAI_API_KEY"],
			RlOpts: ratelimiter.Opts{
				PerUserLimit: 1,
				GlobalLimit:  5,
			},
		},
	)

	eventDispatcher, err := dispatcher.NewEventDispatcher(
		dispatcher.Deps{OpenAiClient: openAiClient},
	)
	if err != nil {
		log.Fatalf("failed to init event dispatcher: %s", err)
	}

	bot, err := telegram.NewBot(
		telegram.Deps{EventDispatcher: eventDispatcher},
		telegram.Opts{Token: envFile["TG_TOKEN"]})
	if err != nil {
		log.Fatalf("failed to start bot: %s", err)
	}

	bot.Listen(ctx)
}
