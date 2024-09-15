package openai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sashabaranov/go-openai"
)

type Opts struct {
	ApiKey string
}

type Client struct {
	api *openai.Client
}

func NewClient(opts Opts) *Client {
	client := openai.NewClient(opts.ApiKey)

	return &Client{
		api: client,
	}
}

type CompleteChatData struct {
	User    string
	Content string
}

func (c *Client) CompleteChat(ctx context.Context, d *CompleteChatData) (string, error) {
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

	choices := resp.Choices
	if len(choices) > 0 {
		return choices[0].Message.Content, nil
	}

	return "No choice to answer :(", nil
}
