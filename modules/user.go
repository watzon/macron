package modules

import (
	"fmt"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
	"github.com/watzon/hdur"
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
	m.AddCommand(mute)
	m.AddCommand(unmute)
	m.AddCommand(unban)
	m.AddCommand(kick)

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
		basicUser, err := utilities.GetUserFromArgs(ctx, args)
		if err != nil {
			return err
		}

		// Get ban duration if provided
		duration := args.GetDuration("duration")

		chat := u.EffectiveChat()

		// Ban the user
		err = utilities.BanUser(ctx, chat.GetID(), basicUser.ID, duration)
		if err != nil {
			return fmt.Errorf("failed to ban user: %w", err)
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

var mute = command.NewCommand("mute").
	WithUsage("mute [username/id]").
	WithDescription("Mute a user in the chat using their username/ID or by replying to a message.").
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
			Description: "Duration of the mute (eg: 3d, 1w, 7m)",
		},
		command.ArgumentDefinition{
			Name:        "reply",
			Type:        command.TypeReply,
			Required:    false,
			Description: "Reply to a message to mute its sender",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get user from argument or reply
		basicUser, err := utilities.GetUserFromArgs(ctx, args)
		if err != nil {
			return err
		}

		// Get mute duration if provided
		duration := args.GetDuration("duration")

		chat := u.EffectiveChat()

		var untilDate int64
		if !duration.IsZero() {
			untilDate = duration.Add(time.Now()).Unix()
		}

		// Mute the user
		err = utilities.RestrictUser(ctx, chat.GetID(), basicUser.ID, tg.ChatBannedRights{
			UntilDate:    int(untilDate),
			ViewMessages: true,
			SendMessages: true,
			SendMedia:    true,
			SendStickers: true,
			SendGifs:     true,
			SendGames:    true,
			SendInline:   true,
			EmbedLinks:   true,
			SendPolls:    true,
		})
		if err != nil {
			return fmt.Errorf("failed to mute user: %w", err)
		}

		// Send confirmation message
		durationText := "permanently"
		if !duration.IsZero() {
			durationText = fmt.Sprintf("for %s", duration.String())
		}
		_, err = ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("✅ Muted %s %s", utilities.FormatUserName(basicUser), durationText)), &ext.ReplyOpts{})
		return err
	})

var unmute = command.NewCommand("unmute").
	WithUsage("unmute [username/id]").
	WithDescription("Unmute a user in the chat using their username/ID or by replying to a message.").
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
			Description: "Reply to a message to unmute its sender",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get user from argument or reply
		basicUser, err := utilities.GetUserFromArgs(ctx, args)
		if err != nil {
			return err
		}

		chat := u.EffectiveChat()

		// Unmute the user
		err = utilities.RestrictUser(ctx, chat.GetID(), basicUser.ID, tg.ChatBannedRights{
			UntilDate: 0,
		})
		if err != nil {
			return fmt.Errorf("failed to unmute user: %w", err)
		}

		// Send confirmation message
		_, err = ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("✅ Unmuted %s", utilities.FormatUserName(basicUser))), &ext.ReplyOpts{})
		return err
	})

var unban = command.NewCommand("unban").
	WithUsage("unban [username/id]").
	WithDescription("Unban a user from the chat using their username/ID or by replying to a message.").
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
			Description: "Reply to a message to unban its sender",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get user from argument or reply
		basicUser, err := utilities.GetUserFromArgs(ctx, args)
		if err != nil {
			return err
		}

		chat := u.EffectiveChat()

		// Unban the user
		err = utilities.UnbanUser(ctx, chat.GetID(), basicUser.ID)
		if err != nil {
			return fmt.Errorf("failed to unban user: %w", err)
		}

		// Send confirmation message
		_, err = ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("✅ Unbanned %s", utilities.FormatUserName(basicUser))), &ext.ReplyOpts{})
		return err
	})

var kick = command.NewCommand("kick").
	WithUsage("kick [username/id]").
	WithDescription("Kick a user from the chat using their username/ID or by replying to a message.").
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
			Description: "Reply to a message to kick its sender",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		// Get user from argument or reply
		basicUser, err := utilities.GetUserFromArgs(ctx, args)
		if err != nil {
			return err
		}

		chat := u.EffectiveChat()

		// Ban the user
		err = utilities.BanUser(ctx, chat.GetID(), basicUser.ID, hdur.Duration{})
		if err != nil {
			return fmt.Errorf("failed to ban user: %w", err)
		}

		// Unban the user
		err = utilities.UnbanUser(ctx, chat.GetID(), basicUser.ID)
		if err != nil {
			return fmt.Errorf("failed to unban user: %w", err)
		}

		// Send confirmation message
		_, err = ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("✅ Kicked %s", utilities.FormatUserName(basicUser))), &ext.ReplyOpts{})
		return err
	})
