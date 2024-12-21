package modules

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
	"github.com/watzon/macron/command"
	"github.com/watzon/macron/utilities"
)

// UserModule contains user-related commands
type UserModule struct {
	*command.BaseModule
}

// NewUserModule creates a new user module
func NewUserModule() *UserModule {
	m := &UserModule{
		BaseModule: command.NewBaseModule(
			"user",
			"User-related commands for getting user information",
		),
	}

	// Add commands to the module
	m.AddCommand(user)

	return m
}

// Load registers all module commands with the dispatcher
func (m *UserModule) Load(d dispatcher.Dispatcher, prefix string) {
	m.BaseModule.Load(d, prefix)
}

var user = command.NewCommand("user").
	WithUsage("user [username/id]").
	WithDescription("Get information about a Telegram user. Reply to a message to get info about its sender, or specify a username/ID.").
	WithAliases("u").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "user",
			Type:        command.TypeEntity,
			Kind:        command.KindPositional,
			Required:    false,
			Description: "Username or ID of the user (optional if replying to a message)",
		},
		command.ArgumentDefinition{
			Name:        "reply",
			Type:        command.TypeReply,
			Required:    false,
			Description: "Reply to a message to get info about its sender",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		fmt.Println(u.EffectiveMessage.ReplyToMessage)

		// Get user from argument or reply
		var user *types.User
		var err error

		// Check if a user argument was provided
		if args.GetPositionalEntity(0) != "" {
			// Try to resolve user from the provided argument
			user, err = utilities.ResolveUser(ctx, args.GetPositionalEntity(0))
			if err != nil {
				return fmt.Errorf("failed to resolve user: %v", err)
			}
		} else if args.Reply != nil {
			// Try to get the user from the replied message
			if args.Reply.Message != nil {
				// Try to resolve using the message's FromID
				fromID := args.Reply.Message.FromID
				if fromID != nil {
					// Type assert to get the specific peer type
					if peerUser, ok := fromID.(*tg.PeerUser); ok {
						// Try to resolve using the user ID
						if chat, err := ctx.ResolveUsername(fmt.Sprint(peerUser.UserID)); err == nil {
							if u, ok := chat.(*types.User); ok {
								user = u
							} else {
								return fmt.Errorf("sender is not a user")
							}
						} else {
							return fmt.Errorf("could not resolve sender: %v", err)
						}
					} else {
						return fmt.Errorf("sender is not a user")
					}
				} else {
					return fmt.Errorf("message has no sender ID")
				}
			} else {
				return fmt.Errorf("could not get replied message")
			}
		} else {
			return fmt.Errorf("please provide a username/ID or reply to a message")
		}

		// Build a nicely formatted message with user information
		info := "👤 **User Information**\n\n"
		info += fmt.Sprintf("**ID:** `%d`\n", user.ID)

		if user.Username != "" {
			info += fmt.Sprintf("**Username:** @%s\n", user.Username)
		}

		if user.FirstName != "" {
			info += fmt.Sprintf("**First Name:** %s\n", user.FirstName)
		}

		if user.LastName != "" {
			info += fmt.Sprintf("**Last Name:** %s\n", user.LastName)
		}

		info += fmt.Sprintf("**Bot:** %v\n", user.Bot)
		info += fmt.Sprintf("**Verified:** %v\n", user.Verified)
		info += fmt.Sprintf("**Scam:** %v\n", user.Scam)
		info += fmt.Sprintf("**Fake:** %v\n", user.Fake)
		info += fmt.Sprintf("**Premium:** %v\n", user.Premium)

		// Send the formatted message using StylizeText for markdown parsing
		_, err = ctx.Reply(u, ext.ReplyTextStyledTextArray(parsemode.StylizeText(info)), &ext.ReplyOpts{})
		return err
	})
