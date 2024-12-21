package modules

import (
	"fmt"
	"time"

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
	m.AddCommand(ban)

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
			Name:        "id",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Get the user's ID",
		},
		command.ArgumentDefinition{
			Name:        "mention",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Mention the user in the reply (otherwise wraps the username with monospace)",
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
		var basicUser *types.User
		var userFull *tg.UserFull
		var err error

		// Check if a user argument was provided
		if args.GetPositionalEntity(0) != "" {
			// Try to resolve user from the provided argument
			basicUser, err = utilities.ResolveUser(ctx, args.GetPositionalEntity(0))
			if err != nil {
				return fmt.Errorf("failed to resolve user: %v", err)
			}
		} else if args.Reply != nil {
			// Try to get the user from the replied message
			if args.Reply.Message != nil {
				basicUser, err = utilities.UserFromMessage(ctx, args.Reply.Message)
				if err != nil {
					return fmt.Errorf("failed to get user from replied message: %v", err)
				}
			} else {
				return fmt.Errorf("could not get replied message")
			}
		} else {
			return fmt.Errorf("please provide a username/ID or reply to a message")
		}

		if args.GetBool("id") {
			// Send the user ID as a reply
			info := "👤 *User Information*\n"
			info += " └─ *Basic Info*\n"
			info += fmt.Sprintf("    └─ *ID:* `%d`\n", basicUser.ID)
			_, err = ctx.Reply(u, ext.ReplyTextStyledTextArray(parsemode.StylizeText(info)), &ext.ReplyOpts{})
			return err
		}

		info := "👤 *User Information*\n"
		info += " ├─ *Basic Info*\n"
		info += fmt.Sprintf(" │  ├─ *ID:* `%d`\n", basicUser.ID)

		// Get full user info
		userFull, err = ctx.GetUser(basicUser.ID)
		if err != nil {
			return fmt.Errorf("failed to get full user info: %v", err)
		}

		if basicUser.Username != "" {
			var username string
			if args.GetBool("mention") {
				username = fmt.Sprintf("@%s", basicUser.Username)
			} else {
				username = fmt.Sprintf("`@%s`", basicUser.Username)
			}
			info += fmt.Sprintf(" │  ├─ *Username:* %s\n", username)
		}

		if basicUser.FirstName != "" {
			info += fmt.Sprintf(" │  ├─ *First Name:* %s\n", basicUser.FirstName)
		}

		if basicUser.LastName != "" {
			info += fmt.Sprintf(" │  └─ *Last Name:* %s\n", basicUser.LastName)
		}

		info += " ├─ *Status*\n"
		info += fmt.Sprintf(" │  ├─ *Bot:* %v\n", basicUser.Bot)
		info += fmt.Sprintf(" │  ├─ *Verified:* %v\n", basicUser.Verified)
		info += fmt.Sprintf(" │  ├─ *Premium:* %v\n", basicUser.Premium)
		info += fmt.Sprintf(" │  ├─ *Scam:* %v\n", basicUser.Scam)
		info += fmt.Sprintf(" │  └─ *Fake:* %v\n", basicUser.Fake)

		// Add additional information section
		info += " └─ *Additional Info*\n"

		if about, ok := userFull.GetAbout(); ok && about != "" {
			info += fmt.Sprintf("    ├─ *Bio:* %s\n", about)
		}

		// Add last seen status if available
		info += fmt.Sprintf("    ├─ *Last Seen:* %s\n", utilities.FormatUserStatus(basicUser.Status))

		// Add other useful info
		info += fmt.Sprintf("    ├─ *Phone Calls Available:* %v\n", userFull.PhoneCallsAvailable)
		info += fmt.Sprintf("    ├─ *Phone Calls Private:* %v\n", userFull.PhoneCallsPrivate)
		info += fmt.Sprintf("    ├─ *Can Pin Message:* %v\n", userFull.CanPinMessage)
		info += fmt.Sprintf("    └─ *Common Chats Count:* %v\n", userFull.CommonChatsCount)

		// Send the formatted message using StylizeText for markdown parsing
		_, err = ctx.Reply(u, ext.ReplyTextStyledTextArray(parsemode.StylizeText(info)), &ext.ReplyOpts{})
		return err
	})

var ban = command.NewCommand("ban").
	WithUsage("ban [username/id]").
	WithDescription("Ban a user from the chat using their username/ID or by replying to a message.").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "user",
			Type:        command.TypeEntity,
			Kind:        command.KindPositional,
			Required:    false,
			Description: "Username or ID of the user (optional if replying to a message)",
		},
		command.ArgumentDefinition{
			Name:        "duration",
			Type:        command.TypeDuration,
			Kind:        command.KindNamed,
			Required:    false,
			Description: "Duration of the ban (eg: 3d, 1w, 7m)",
		},
		command.ArgumentDefinition{
			Name:        "silent",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Ban the user silently (no notification and deleting the message)",
		},
		command.ArgumentDefinition{
			Name:        "delete",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Delete the replied to message",
		},
		command.ArgumentDefinition{
			Name:        "spam",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Report the user for spam",
		},
		command.ArgumentDefinition{
			Name:        "reply",
			Type:        command.TypeReply,
			Required:    false,
			Description: "Reply to a message to ban its sender",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get user from argument or reply
		var basicUser *types.User
		var err error

		// Check if a user argument was provided
		if args.GetPositionalEntity(0) != "" {
			// Try to resolve user from the provided argument
			basicUser, err = utilities.ResolveUser(ctx, args.GetPositionalEntity(0))
			if err != nil {
				return fmt.Errorf("failed to resolve user: %v", err)
			}
		} else if args.Reply != nil {
			// Try to get the user from the replied message
			if args.Reply.Message != nil {
				basicUser, err = utilities.UserFromMessage(ctx, args.Reply.Message)
				if err != nil {
					return fmt.Errorf("failed to get user from replied message: %v", err)
				}
			} else {
				return fmt.Errorf("could not get replied message")
			}
		} else {
			return fmt.Errorf("please provide a username/ID or reply to a message")
		}

		// Get ban duration if provided
		duration := args.GetDuration("duration")
		var untilDate int64
		if !duration.IsZero() {
			untilDate = duration.Add(time.Now()).Unix()
		}

		chat := u.EffectiveChat()

		// Ban the user
		_, err = ctx.BanChatMember(chat.GetID(), basicUser.ID, int(untilDate))
		if err != nil {
			return fmt.Errorf("failed to ban user: %v", err)
		}

		if args.GetBool("delete") {
			// Delete the replied message
			err = ctx.DeleteMessages(chat.GetID(), []int{u.EffectiveMessage.ID})
			if err != nil {
				return fmt.Errorf("failed to delete message: %v", err)
			}
		}

		if args.GetBool("silent") {
			// Delete the sent message
			err = ctx.DeleteMessages(chat.GetID(), []int{u.EffectiveMessage.ID})
			if err != nil {
				return fmt.Errorf("failed to delete message: %v", err)
			}
		} else {
			// Send confirmation message
			durationText := "permanently"
			if !duration.IsZero() {
				durationText = fmt.Sprintf("for %s", duration.String())
			}
			_, err = ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("✅ Banned %s %s", utilities.FormatUserName(basicUser), durationText)), &ext.ReplyOpts{})
			return err
		}

		if args.GetBool("spam") {

		}

		return nil
	})
