package modules

import (
	"fmt"
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/watzon/macron/command"
)

// MiscModule contains miscellaneous utility commands
type MiscModule struct {
	*command.BaseModule
}

// NewMiscModule creates a new misc module
func NewMiscModule() *MiscModule {
	m := &MiscModule{
		BaseModule: command.NewBaseModule(
			"misc",
			"Miscellaneous utility commands like ping and echo",
		),
	}

	// Add commands to the module
	m.AddCommand(ping)
	m.AddCommand(echo)

	return m
}

// Load registers all module commands with the dispatcher
func (m *MiscModule) Load(d dispatcher.Dispatcher, prefix string) {
	m.BaseModule.Load(d, prefix)
}

var ping = command.NewCommand("ping").
	WithUsage("ping").
	WithDescription("Responds with pong, optionally multiple times").
	WithHandler(func(ctx *ext.Context, u *ext.Update, _ *command.Arguments) error {
		start := time.Now()
		msg, err := ctx.Reply(u, ext.ReplyTextString("pong"), &ext.ReplyOpts{})
		if err != nil {
			return err
		}

		rtt := time.Since(start)
		_, err = ctx.EditMessage(u.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      msg.ID,
			Peer:    u.EffectiveChat().GetInputPeer(),
			Message: fmt.Sprintf("Pong (%.2fms)", float64(rtt.Microseconds())/1000.0),
		})
		return err
	})

var echo = command.NewCommand("echo").
	WithUsage("echo <text> [-repeat N] [-uppercase] [...words]").
	WithDescription("Echoes back text with optional modifications").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "text",
			Type:        command.TypeString,
			Kind:        command.KindPositional,
			Required:    true,
			Description: "The main text to echo",
		},
		command.ArgumentDefinition{
			Name:        "repeat",
			Type:        command.TypeInt,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     1,
			Description: "Number of times to repeat the message",
		},
		command.ArgumentDefinition{
			Name:        "uppercase",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Convert the text to uppercase",
		},
		command.ArgumentDefinition{
			Name:        "words",
			Type:        command.TypeString,
			Kind:        command.KindVariadic,
			Required:    false,
			Description: "Additional words to append",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get the main text (positional argument)
		text := args.GetPositionalString(0)
		if text == "" {
			return fmt.Errorf("text argument is required")
		}

		// Get named arguments
		repeat := args.GetInt("repeat")
		if repeat == 0 {
			repeat = 1 // Default to 1 if not provided or invalid
		}
		uppercase := args.GetBool("uppercase")

		// Get variadic arguments
		extraWords := args.GetVariadic()

		// Build the message
		message := text
		if len(extraWords) > 0 {
			// Convert []interface{} to []string and filter out empty/invalid values
			var words []string
			for _, w := range extraWords {
				if s, ok := w.(string); ok && s != "" {
					words = append(words, s)
				}
			}
			if len(words) > 0 {
				message += " " + strings.Join(words, " ")
			}
		}

		if uppercase {
			message = strings.ToUpper(message)
		}

		// Send the message the specified number of times
		for i := 0; i < repeat; i++ {
			_, err := ctx.Reply(u, ext.ReplyTextString(message), &ext.ReplyOpts{})
			if err != nil {
				return fmt.Errorf("failed to send message: %v", err)
			}
		}

		return nil
	})
