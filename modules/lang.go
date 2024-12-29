package modules

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/functions"
	"github.com/celestix/gotgproto/parsemode"
	"github.com/gotd/td/tg"
	"github.com/watzon/macron/command"
	"github.com/watzon/macron/config"
	"github.com/watzon/macron/logger"
	"github.com/watzon/macron/services"
	"github.com/watzon/macron/utilities"
)

// LangModule contains language-related commands
type LangModule struct {
	*command.BaseModule
}

var llmService *services.LLMService

// NewLangModule creates a new lang module
func NewLangModule() *LangModule {
	m := &LangModule{
		BaseModule: command.NewBaseModule(
			"lang",
			"Language-related commands like translate",
		),
	}

	// Add commands to the module

	// Add translate command only if we have an OpenRouter API key defined
	openRouterAPIKey := config.Instance().OpenRouterAPIKey
	if openRouterAPIKey != "" {
		llmService = services.NewLLMService(openRouterAPIKey)
		m.AddCommand(translate)
	} else {
		fmt.Println("OpenRouter API key not provided, so the translate command will be disabled")
	}

	return m
}

// Load registers all module commands with the dispatcher
func (m *LangModule) Load(d dispatcher.Dispatcher, prefix string) {
	m.BaseModule.Load(d, prefix)
}

var translate = command.NewCommand("translate").
	WithUsage("translate [-to <target_language>] <text>").
	WithDescription("Translates text to another language").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "to",
			Type:        command.TypeString,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     "English",
			Description: "The target language for translation",
		},
		command.ArgumentDefinition{
			Name:        "count",
			Type:        command.TypeInt,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     1,
			Description: "Number of messages to translate (including replied message)",
		},
		command.ArgumentDefinition{
			Name:        "silent",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Send the translation to the log channel and delete the command message",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		var messages []struct {
			From string
			Text string
		}

		count := args.GetInt("count")
		if count < 1 {
			count = 1
		}

		// If there's a direct text argument, use it as the only message
		text := args.GetRestString()
		if text != "" {
			from := "Unknown User"
			if u.EffectiveMessage.Message != nil {
				user, err := utilities.UserFromMessage(ctx, u.EffectiveMessage.Message)
				if err == nil && user != nil {
					from = utilities.FormatUserName(user)
				}
			}
			messages = append(messages, struct {
				From string
				Text string
			}{
				From: from,
				Text: text,
			})
		} else {
			// Start with the replied message
			replyMsg := u.EffectiveMessage.ReplyToMessage
			if replyMsg == nil || replyMsg.Message == nil {
				return fmt.Errorf("text argument is required or reply to a message")
			}

			// Get the chat ID and message ID to start from
			chatId := utilities.GetEffectiveChatID(u)
			startMsgID := replyMsg.Message.GetID()

			// Get the peer for the chat
			peer := ctx.PeerStorage.GetInputPeerById(chatId)
			if peer == nil {
				return fmt.Errorf("failed to get peer for chat")
			}

			// Create a request to get message history
			req := &tg.MessagesGetHistoryRequest{
				Peer:      peer,
				OffsetID:  startMsgID,
				AddOffset: -1,
				Limit:     count,
			}

			// Get the message history
			history, err := ctx.Raw.MessagesGetHistory(ctx.Context, req)
			if err != nil {
				return fmt.Errorf("failed to get message history: %v", err)
			}

			// Extract messages from the response
			var historyMsgs []tg.MessageClass
			switch hist := history.(type) {
			case *tg.MessagesMessages:
				historyMsgs = hist.Messages
			case *tg.MessagesMessagesSlice:
				historyMsgs = hist.Messages
			case *tg.MessagesChannelMessages:
				historyMsgs = hist.Messages
			default:
				return fmt.Errorf("unexpected response type from GetHistory: %T", history)
			}

			// Process the messages
			for _, msg := range historyMsgs {
				if m, ok := msg.(*tg.Message); ok && m.Message != "" {
					from := "Unknown User"

					// Check if the message is forwarded
					if fwd, ok := m.GetFwdFrom(); ok {
						// For forwarded messages, try to get the name directly from the header
						if name, ok := fwd.GetFromName(); ok && name != "" {
							from = name
						} else if author, ok := fwd.GetPostAuthor(); ok && author != "" {
							from = author
						} else if fromID, ok := fwd.GetFromID(); ok {
							// Try to get the user ID from the forwarded message
							if peerUser, ok := fromID.(*tg.PeerUser); ok {
								// Try to get input peer from ID
								peer := functions.GetInputPeerClassFromId(ctx.PeerStorage, peerUser.UserID)
								if peer != nil {
									if inputUser, ok := peer.(*tg.InputPeerUser); ok {
										users, err := ctx.Raw.UsersGetUsers(ctx.Context, []tg.InputUserClass{
											&tg.InputUser{
												UserID:     inputUser.UserID,
												AccessHash: inputUser.AccessHash,
											},
										})
										if err == nil && len(users) > 0 {
											if user, ok := users[0].(*tg.User); ok {
												from = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
											}
										}
									}
								}
								if from == "" {
									from = fmt.Sprintf("User %d", peerUser.UserID)
								}
							} else if _, ok := fromID.(*tg.PeerChannel); ok {
								// It's forwarded from a channel, try to get the post author
								if author, ok := fwd.GetPostAuthor(); ok && author != "" {
									from = author + " (Channel)"
								} else {
									from = "Channel"
								}
							}
						}
					} else {
						// Not forwarded, get the original sender
						if fromID, ok := m.GetFromID(); ok {
							if _, ok := fromID.(*tg.PeerUser); ok {
								if user, err := utilities.UserFromMessage(ctx, m); err == nil && user != nil {
									from = utilities.FormatUserName(user)
								}
							}
						}
					}

					messages = append([]struct {
						From string
						Text string
					}{{
						From: from,
						Text: strings.TrimSpace(m.Message),
					}}, messages...)
				}
			}
		}

		if len(messages) == 0 {
			return fmt.Errorf("no messages to translate")
		}

		// Build conversation text
		var conversationText strings.Builder
		for i, msg := range messages {
			if i > 0 {
				conversationText.WriteString("\n\n")
			}
			conversationText.WriteString(fmt.Sprintf("%s:\n%s", msg.From, msg.Text))
		}

		targetLanguage := args.GetString("to")
		if llmService == nil {
			_, err := ctx.Reply(u, ext.ReplyTextString("LLM service not initialized"), &ext.ReplyOpts{})
			return err
		}

		translatedText, err := llmService.TranslateText(ctx.Context, conversationText.String(), targetLanguage)
		if err != nil {
			_, err := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Error translating text: %v", err)), &ext.ReplyOpts{})
			return err
		}

		// Delete the command message
		chatId := utilities.GetEffectiveChatID(u)
		err = ctx.DeleteMessages(chatId, []int{u.EffectiveMessage.GetID()})

		if args.GetBool("silent") {
			msg := fmt.Sprintf("🌐 *Translation result*\n\n*Input:*\n`%s`\n\n*Output:*\n`%s`", conversationText.String(), translatedText)
			logger.Log(msg)
			return err
		} else {
			msg := fmt.Sprintf("|%s|\n\n*%s*", conversationText.String(), strings.TrimSpace(translatedText))
			_, err = ctx.Reply(u, ext.ReplyTextStyledTextArray(parsemode.StylizeText(msg)), &ext.ReplyOpts{})
			return err
		}
	})
