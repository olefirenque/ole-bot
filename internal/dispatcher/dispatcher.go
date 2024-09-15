package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/agnivade/levenshtein"
	"github.com/dghubble/trie"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"ole-bot/internal/openai"
)

// Deps is a carrier of dependencies for event dispatcher.
type Deps struct {
	OpenAiClient *openai.Client
}

// EventDispatcher dispatches commands according to command prefixes and other heuristics.
// Command inputs are handled via fuzzy search.
type EventDispatcher struct {
	openAiClient *openai.Client

	commandTrie atomic.Pointer[trie.RuneTrie]
}

// NewEventDispatcher creates EventDispatcher instance with built command trie.
func NewEventDispatcher(deps Deps) (*EventDispatcher, error) {
	ed := &EventDispatcher{
		openAiClient: deps.OpenAiClient,
	}
	if err := ed.buildTrie(); err != nil {
		return nil, err
	}

	return ed, nil
}

func (ed *EventDispatcher) buildTrie() error {
	commandTrie := trie.NewRuneTrie()
	for _, command := range commands {
		commandTrie.Put(string(command), command)
	}

	ed.commandTrie.Store(commandTrie)
	return nil
}

func (ed *EventDispatcher) DispatchMessage(ctx context.Context, message *tgbotapi.Message) string {
	if message.IsCommand() {
		return ed.handleCommand(ctx, message)
	}

	return ""
}

func (ed *EventDispatcher) handleCommand(ctx context.Context, message *tgbotapi.Message) string {
	parsedCommands, exact := ed.getRelevantCommands(message.CommandWithAt())
	if !exact {
		return handleIncorrectCommand(ctx, parsedCommands)
	}
	command := parsedCommands[0]

	if reply, ok := constantReplies[command]; ok {
		return reply
	}
	message.CommandArguments()

	switch command {
	case SingleGptMessageCommand:
		response, err := ed.openAiClient.CompleteChat(ctx, &openai.CompleteChatData{
			User:    message.Chat.UserName,
			Content: message.CommandArguments(),
		})
		if err != nil {
			return fmt.Sprintf("Не удалось отправить сообщение \n(%s)", err)
		}

		return response
	}

	return ""
}

func handleIncorrectCommand(_ context.Context, parsedCommands []Command) string {
	if len(parsedCommands) == 0 {
		return fmt.Sprintf("Не понимаю команду")
	}

	return fmt.Sprintf("Возможно вы имели в виду что-то из этого: %s", strings.Join(commandList(parsedCommands), ", "))
}

func (ed *EventDispatcher) getRelevantCommands(command string) ([]Command, bool) {
	ct := ed.commandTrie.Load()
	if x := ct.Get(command); x != nil {
		return []Command{x.(Command)}, true
	}

	const maxDistance = 3
	var closestCommands []Command
	_ = ct.Walk(func(key string, value any) error {
		c := value.(Command)
		distance := levenshtein.ComputeDistance(command, key)
		if distance < maxDistance || strings.HasPrefix(key, command) {
			closestCommands = append(closestCommands, c)
		}
		return nil
	})

	return closestCommands, false
}
