package modules

import (
	"bytes"
	"fmt"
	"image/png"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode"

	cowsay "github.com/Code-Hex/Neo-cowsay"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gotgbotext "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	messagefilters "github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
	"github.com/watzon/macron/command"
	"github.com/watzon/macron/utilities"
)

type botInstance struct {
	bot     *gotgbot.Bot
	updater *gotgbotext.Updater
}

var (
	activeBots = make(map[string]*botInstance)
	botsLock   sync.RWMutex
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
	m.AddCommand(botbot)
	m.AddCommand(screenshot)

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

var botbot = command.NewCommand("botbot").
	WithUsage("botbot [-stop BOT_USERNAME] [-delete BOT_USERNAME] <bot_token>").
	WithDescription("Spin up a new bot instance, or manage existing bots").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "stop",
			Type:        command.TypeString,
			Kind:        command.KindNamed,
			Required:    false,
			Description: "Stop a running bot by username",
		},
		command.ArgumentDefinition{
			Name:        "tail",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Tail the bot's logs, sending them to wherever the botbot command was run from",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Handle stop flag
		if stopBot := args.GetString("stop"); stopBot != "" {
			botsLock.Lock()
			defer botsLock.Unlock()

			if instance, exists := activeBots[stopBot]; exists {
				instance.updater.Stop()
				delete(activeBots, stopBot)
				_, err := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Bot @%s has been stopped", stopBot)), nil)
				return err
			}
			_, err := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Bot @%s not found", stopBot)), nil)
			return err
		}

		// Handle new bot creation
		token := args.GetRestString()
		if token == "" {
			_, err := ctx.Reply(u, ext.ReplyTextString("Please provide a bot token"), nil)
			return err
		}

		// Create new bot instance with BotOpts
		bot, err := gotgbot.NewBot(token, &gotgbot.BotOpts{})
		if err != nil {
			_, err := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Failed to create bot: %v", err)), nil)
			return err
		}

		// Create dispatcher first
		dispatcher := gotgbotext.NewDispatcher(&gotgbotext.DispatcherOpts{
			// Error handler
			Error: func(b *gotgbot.Bot, ctx *gotgbotext.Context, err error) gotgbotext.DispatcherAction {
				fmt.Printf("Error handling update: %v\n", err)
				return gotgbotext.DispatcherActionNoop
			},
		})

		// Add /help command handler to dispatcher
		dispatcher.AddHandler(handlers.NewCommand("help", func(b *gotgbot.Bot, ctx *gotgbotext.Context) error {
			_, err := ctx.EffectiveMessage.Reply(b, "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", &gotgbot.SendMessageOpts{})
			return err
		}))

		// Check if tail flag is set. If so log all incoming updates to the EffectiveChat
		if args.GetBool("tail") {
			dispatcher.AddHandler(handlers.NewMessage(messagefilters.All, func(b *gotgbot.Bot, ctx *gotgbotext.Context) error {
				fmt.Printf("Received update: %+v\n", ctx.EffectiveMessage)
				return nil
			}))
		}

		// Create updater with dispatcher
		updater := gotgbotext.NewUpdater(dispatcher, &gotgbotext.UpdaterOpts{
			ErrorLog: nil,
			UnhandledErrFunc: func(err error) {
				fmt.Printf("Unhandled error: %v\n", err)
			},
		})

		// Start receiving updates
		err = updater.StartPolling(bot, &gotgbotext.PollingOpts{})
		if err != nil {
			_, err := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Failed to start bot: %v", err)), nil)
			return err
		}

		// Store both bot and updater in active bots map
		botsLock.Lock()
		activeBots[bot.User.Username] = &botInstance{
			bot:     bot,
			updater: updater,
		}
		botsLock.Unlock()

		_, err = ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Bot @%s is now running!", bot.User.Username)), nil)
		return err
	})

var screenshot = command.NewCommand("screenshot").
	WithUsage("screenshot [-count N]").
	WithDescription("Creates a fake screenshot of messages. If -count is provided, includes N messages before the replied message. Otherwise only shows the replied message.").
	WithAliases("sc").
	WithOutgoing(true).
	WithIncoming(false).
	WithArguments(
		command.ArgumentDefinition{
			Name:        "count",
			Type:        command.TypeInt,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     0,
			Description: "Number of messages to include before the replied message",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		if u.EffectiveMessage.ReplyToMessage == nil {
			_, err := ctx.Reply(u, ext.ReplyTextString("Please reply to a message to create a screenshot"), &ext.ReplyOpts{})
			return err
		}

		count := args.GetInt("count")

		// Collect messages and their data
		var messages []utilities.MessageData

		// Add the replied message first
		replyMsg := u.EffectiveMessage.ReplyToMessage
		if replyMsg.Message != nil {
			user, err := utilities.UserFromMessage(ctx, replyMsg.Message)
			if err != nil {
				user = &types.User{FirstName: "Unknown User"}
			}

			// Get user avatar
			avatar, err := utilities.GetUserAvatar(ctx, user)
			if err != nil {
				// Continue without avatar if we can't get it
				fmt.Printf("Failed to get avatar for user %s: %v\n", utilities.FormatUserName(user), err)
			}

			messages = append(messages, utilities.MessageData{
				User:      user,
				Text:      replyMsg.Message.Message,
				Entities:  replyMsg.Message.Entities,
				Avatar:    avatar,
				Timestamp: int64(replyMsg.Message.Date),
			})
		}

		// Get previous messages if count > 0
		if count > 0 {
			// Get message history
			history, err := ctx.Raw.MessagesGetHistory(ctx.Context, &tg.MessagesGetHistoryRequest{
				Peer:      u.EffectiveChat().GetInputPeer(),
				OffsetID:  replyMsg.Message.ID,
				AddOffset: 0,
				Limit:     count,
			})
			if err != nil {
				return fmt.Errorf("failed to get message history: %w", err)
			}

			switch m := history.(type) {
			case *tg.MessagesChannelMessages:
				for _, msg := range m.Messages {
					if msg, ok := msg.(*tg.Message); ok {
						if msg.Message == "" {
							continue
						}
						user, err := utilities.UserFromMessage(ctx, msg)
						if err != nil {
							user = &types.User{FirstName: "Unknown User"}
						}

						// Get user avatar
						avatar, err := utilities.GetUserAvatar(ctx, user)
						if err != nil {
							// Continue without avatar if we can't get it
							fmt.Printf("Failed to get avatar for user %s: %v\n", utilities.FormatUserName(user), err)
						}

						messages = append([]utilities.MessageData{{
							User:      user,
							Text:      msg.Message,
							Entities:  msg.Entities,
							Avatar:    avatar,
							Timestamp: int64(msg.Date),
						}}, messages...)
					}
				}
			case *tg.MessagesMessages:
				for _, msg := range m.Messages {
					if msg, ok := msg.(*tg.Message); ok {
						if msg.Message == "" {
							continue
						}
						user, err := utilities.UserFromMessage(ctx, msg)
						if err != nil {
							user = &types.User{FirstName: "Unknown User"}
						}

						// Get user avatar
						avatar, err := utilities.GetUserAvatar(ctx, user)
						if err != nil {
							// Continue without avatar if we can't get it
							fmt.Printf("Failed to get avatar for user %s: %v\n", utilities.FormatUserName(user), err)
						}

						messages = append([]utilities.MessageData{{
							User:      user,
							Text:      msg.Message,
							Entities:  msg.Entities,
							Avatar:    avatar,
							Timestamp: int64(msg.Date),
						}}, messages...)
					}
				}
			}
		}

		// Generate the screenshot
		style := utilities.DefaultMessageStyle()
		img, err := utilities.GenerateMessageScreenshot(messages, style)
		if err != nil {
			return fmt.Errorf("failed to generate screenshot: %v", err)
		}

		// Convert image to bytes
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			return fmt.Errorf("failed to encode image: %v", err)
		}

		// Create a random ID for the file
		fileID := rand.Int63()

		// Upload the file data first
		uploaded, err := ctx.Raw.UploadSaveFilePart(ctx.Context, &tg.UploadSaveFilePartRequest{
			FileID:   fileID,
			FilePart: 0,
			Bytes:    buf.Bytes(),
		})
		if err != nil {
			return fmt.Errorf("failed to upload file: %w", err)
		}
		if !uploaded {
			return fmt.Errorf("failed to upload file: server returned false")
		}

		// Send the image
		_, err = ctx.SendMedia(u.EffectiveChat().GetID(), &tg.MessagesSendMediaRequest{
			Media: &tg.InputMediaUploadedPhoto{
				File: &tg.InputFile{
					ID:    fileID,
					Name:  "screenshot.png",
					Parts: 1,
				},
			},
			ReplyTo: &tg.InputReplyToMessage{
				ReplyToMsgID: u.EffectiveMessage.ID,
			},
		})
		return err
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
