package openai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/sashabaranov/go-openai"
	"ole-bot/pkg/ratelimiter"
)

const defaultTimeout = 10 * time.Second

type Opts struct {
	ApiKey   string
	ProxyURL *url.URL

	RlOpts ratelimiter.Opts
}

type Client struct {
	api *openai.Client

	ratelimiter *ratelimiter.Ratelimiter
}

func NewClient(opts Opts) *Client {
	config := openai.DefaultConfig(opts.ApiKey)
	if opts.ProxyURL != nil {
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(opts.ProxyURL),
			},
			Timeout: defaultTimeout,
		}
	}

	client := openai.NewClientWithConfig(config)

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
	ctxErrReply          = "OpenAI не отвечает слишком долго"
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
		slog.ErrorContext(ctx, fmt.Sprintf("openai/Client/CompleteChat: %s", err))
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErrReply, nil
		}
		return "", fmt.Errorf("failed to complete chat: %w", err)
	}

	if choices := resp.Choices; len(choices) > 0 {
		return choices[0].Message.Content, nil
	}

	return noChoiceReply, nil
}
