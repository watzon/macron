package modules

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"

	cowsay "github.com/Code-Hex/Neo-cowsay"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode"
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
			"Miscellaneous utility commands like ping, echo, mock, reverse, leet, vaporwave, and cowsay",
		),
	}

	// Add commands to the module
	m.AddCommand(ping)
	m.AddCommand(echo)
	m.AddCommand(mock)
	m.AddCommand(reverse)
	m.AddCommand(leet)
	m.AddCommand(vaporwave)
	m.AddCommand(cowsayCmd)

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
	WithUsage("echo [-repeat N] [-uppercase] <text>").
	WithDescription("Echoes back text with optional modifications").
	WithArguments(
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
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get named arguments
		repeat := args.GetInt("repeat")
		if repeat == 0 {
			repeat = 1 // Default to 1 if not provided or invalid
		}
		uppercase := args.GetBool("uppercase")

		return handleTextCommand(ctx, u, args, func(text string) string {
			if uppercase {
				text = strings.ToUpper(text)
			}
			var result strings.Builder
			for i := 0; i < repeat; i++ {
				result.WriteString(text)
			}
			return result.String()
		})
	})

var mock = command.NewCommand("mock").
	WithUsage("mock <text>").
	WithDescription("Transforms text to look like the spongebob meme").
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		return handleTextCommand(ctx, u, args, func(text string) string {
			var result strings.Builder
			for _, r := range text {
				if rand.Intn(2) == 0 {
					result.WriteString(strings.ToLower(string(r)))
				} else {
					result.WriteString(strings.ToUpper(string(r)))
				}
			}
			return result.String()
		})
	})

var reverse = command.NewCommand("reverse").
	WithUsage("reverse <text>").
	WithDescription("Reverses the input text").
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		return handleTextCommand(ctx, u, args, func(text string) string {
			var result strings.Builder
			for i := len(text) - 1; i >= 0; i-- {
				result.WriteByte(text[i])
			}
			return result.String()
		})
	})

var leet = command.NewCommand("leet").
	WithUsage("leet <text>").
	WithDescription("Converts text to leetspeak").
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		return handleTextCommand(ctx, u, args, func(text string) string {
			leetMap := map[rune]string{
				'a': "4", 'A': "4",
				'e': "3", 'E': "3",
				'i': "1", 'I': "1",
				'o': "0", 'O': "0",
				't': "7", 'T': "7",
				's': "5", 'S': "5",
			}
			var result strings.Builder
			for _, r := range text {
				if val, ok := leetMap[r]; ok {
					result.WriteString(val)
				} else {
					result.WriteRune(r)
				}
			}
			return result.String()
		})
	})

var vaporwave = command.NewCommand("vaporwave").
	WithUsage("vaporwave <text>").
	WithDescription("Converts text to vaporwave aesthetic").
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		return handleTextCommand(ctx, u, args, func(text string) string {
			var result strings.Builder
			for _, r := range text {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					result.WriteString(string(r + 0xfee0))
				} else {
					result.WriteRune(r)
				}
			}
			return result.String()
		})
	})

var cowsayCmd = command.NewCommand("cowsay").
	WithUsage("cowsay <text>").
	WithDescription("Displays text in a speech bubble from a cow").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "list",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "List the available cow types",
		},
		command.ArgumentDefinition{
			Name:        "cow",
			Type:        command.TypeString,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     "cow",
			Description: "The cow type to use",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		if args.GetBool("list") {
			cows := cowsay.Cows()
			msgBuilder := strings.Builder{}
			msgBuilder.WriteString("🐮 *Available cows:*\n")
			for i, cow := range cows {
				msgBuilder.WriteString(fmt.Sprintf("`%s`", cow))
				if i < len(cows)-1 {
					msgBuilder.WriteString(", ")
				}
			}
			msg := parsemode.StylizeText(msgBuilder.String())
			_, err := ctx.Reply(u, ext.ReplyTextStyledTextArray(msg), &ext.ReplyOpts{})
			return err
		}

		return handleTextCommand(ctx, u, args, func(text string) string {
			cow, err := cowsay.Say(
				cowsay.Phrase(text),
				cowsay.Type(args.GetString("cow")),
			)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return cow
		})
	})

func handleTextCommand(ctx *ext.Context, u *ext.Update, args *command.Arguments, transform func(string) string) error {
	text := args.GetRestString()
	if text == "" {
		if u.EffectiveMessage.ReplyToMessage == nil || u.EffectiveMessage.ReplyToMessage.Message == nil {
			return fmt.Errorf("text argument is required or reply to a message")
		}
		text = u.EffectiveMessage.ReplyToMessage.Text
	}

	transformedText := transform(text)

	if u.EffectiveMessage.ReplyToMessage != nil {
		// Reply to the original message and delete the command message
		_, err := ctx.Reply(u, ext.ReplyTextString(transformedText), &ext.ReplyOpts{
			ReplyToMessageId: u.EffectiveMessage.ReplyToMessage.ID,
		})
		if err != nil {
			return err
		}
		err = ctx.DeleteMessages(u.EffectiveChat().GetID(), []int{u.EffectiveMessage.ID})
		return err
	} else {
		// Edit the current message
		_, err := ctx.EditMessage(u.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      u.EffectiveMessage.ID,
			Peer:    u.EffectiveChat().GetInputPeer(),
			Message: transformedText,
		})
		return err
	}
}
