package openai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sashabaranov/go-openai"
	"ole-bot/pkg/ratelimiter"
)

type Opts struct {
	ApiKey string

	RlOpts ratelimiter.Opts
}

type Client struct {
	api *openai.Client

	ratelimiter *ratelimiter.Ratelimiter
}

func NewClient(opts Opts) *Client {
	client := openai.NewClient(opts.ApiKey)

	return &Client{
		api: client,

		ratelimiter: ratelimiter.NewRatelimiter(opts.RlOpts),
	}
}

type CompleteChatData struct {
	User    string
	Content string
}

const (
	tooManyRequestsReply = "Слишком много запросов к openai :("
	noChoiceReply        = "Нет доступных ответов :("
	emptyUserReply       = "Невозможно сделать запрос для юзера без имени"
	emptyMessageReply    = "Пожалуйста, отправьте непустое сообщение"
)

func (c *Client) CompleteChat(ctx context.Context, d *CompleteChatData) (string, error) {
	if d.User == "" {
		return emptyUserReply, nil
	} else if d.Content == "" {
		return emptyMessageReply, nil
	} else if !c.ratelimiter.Allow(ctx, d.User) {
		return tooManyRequestsReply, nil
	}

	slog.Info("complete chat", "data", d)
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		User:  d.User,
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: d.Content,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete chat: %w", err)
	}

	if choices := resp.Choices; len(choices) > 0 {
		return choices[0].Message.Content, nil
	}

	return noChoiceReply, nil
}
