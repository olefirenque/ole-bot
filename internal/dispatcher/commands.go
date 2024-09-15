package dispatcher

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

type Command string

const (
	HelloCommand            Command = "hello"
	AboutCommand            Command = "about"
	HelpCommand             Command = "help"
	SingleGptMessageCommand Command = "gpt_message"
)

var commands = []Command{
	HelloCommand, AboutCommand, HelpCommand, SingleGptMessageCommand,
}

var constantReplies = map[Command]string{
	HelloCommand: "Ну привет.",
	AboutCommand: "По всем вопросам к @olefirenque (https://t.me/olefirenque).",
	HelpCommand:  fmt.Sprintf("Список команд: \n%s", strings.Join(commandList(commands), "\n")),
}

func commandList(commands []Command) []string {
	return lo.Map(commands, func(c Command, _ int) string {
		return fmt.Sprintf("/%s", c)
	})
}
