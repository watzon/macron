package utilities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/functions"
	"github.com/celestix/gotgproto/storage"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
	"github.com/watzon/hdur"
	"github.com/watzon/macron/command"
)

// SplitCommand splits a command string into command and arguments
func SplitCommand(text string) (string, string) {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// ResolveUser attempts to resolve a user from a username, ID, or other identifier
func ResolveUser(ctx *ext.Context, identifier string) (*types.User, error) {
	// Try username resolution first (with or without @ prefix)
	username := strings.TrimPrefix(identifier, "@")
	if chat, err := ctx.ResolveUsername(username); err == nil {
		if user, ok := chat.(*types.User); ok {
			return user, nil
		}
	}

	// Try as numeric ID
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		// Try to get from peer storage first
		if peer := ctx.PeerStorage.GetPeerById(id); peer.ID != 0 && peer.Type == int(storage.TypeUser) {
			// Try to resolve username to get full user info
			// This will also update the storage with latest info
			if chat, err := ctx.ResolveUsername(peer.Username); err == nil {
				if user, ok := chat.(*types.User); ok {
					return user, nil
				}
			}
		}

		// If not in storage or couldn't resolve, try to get full user info
		_, err := ctx.GetUser(id)
		if err == nil {
			// Try resolving through username again now that we have the user in storage
			if peer := ctx.PeerStorage.GetPeerById(id); peer.ID != 0 && peer.Type == int(storage.TypeUser) && peer.Username != "" {
				if chat, err := ctx.ResolveUsername(peer.Username); err == nil {
					if user, ok := chat.(*types.User); ok {
						return user, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("could not resolve user: %s", identifier)
}

// UserFromMessage attempts to get a User from the message
func UserFromMessage(ctx *ext.Context, m *tg.Message) (*types.User, error) {
	// Try to resolve using the message's FromID
	fromID := m.FromID
	if fromID != nil {
		// Type assert to get the specific peer type
		if peerUser, ok := fromID.(*tg.PeerUser); ok {
			// Try to resolve using the user ID
			return ResolveUser(ctx, fmt.Sprint(peerUser.UserID))
		}
	}

	return nil, fmt.Errorf("could not get user from message")
}

// FormatUserStatus formats a user's status into a human-readable string
func FormatUserStatus(status tg.UserStatusClass) string {
	switch s := status.(type) {
	case *tg.UserStatusOnline:
		return "Online"
	case *tg.UserStatusOffline:
		lastSeen := time.Unix(int64(s.WasOnline), 0)
		return fmt.Sprintf("Last seen %s", lastSeen.Format("2006-01-02 15:04:05"))
	case *tg.UserStatusRecently:
		return "Recently"
	case *tg.UserStatusLastWeek:
		return "Last week"
	case *tg.UserStatusLastMonth:
		return "Last month"
	default:
		return "Long time ago"
	}
}

// FormatUserName formats a user's name using the first and last name if available, then the username, and finally the ID
func FormatUserName(user *types.User) string {
	if user.FirstName != "" && user.LastName != "" {
		return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	} else if user.FirstName != "" {
		return user.FirstName
	} else if user.LastName != "" {
		return user.LastName
	} else if user.Username != "" {
		return fmt.Sprintf("@%s", user.Username)
	}
	return fmt.Sprintf("%d", user.ID)
}

// FillPeerStorage iterates through dialogs and caches them in the peer storage
func FillPeerStorage(client *gotgproto.Client, limit int) error {
	// Get all dialogs
	dialogs, err := client.API().MessagesGetDialogs(context.Background(), &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      limit,
	})
	if err != nil {
		return err
	} else {
		// Process the dialogs response
		switch d := dialogs.(type) {
		case *tg.MessagesDialogs:
			for _, user := range d.Users {
				if u, ok := user.AsNotEmpty(); ok {
					client.PeerStorage.AddPeer(u.ID, u.AccessHash, storage.TypeUser, u.Username)
				}
			}
			for _, chat := range d.Chats {
				switch c := chat.(type) {
				case *tg.Channel:
					client.PeerStorage.AddPeer(c.ID, c.AccessHash, storage.TypeChannel, c.Username)
				case *tg.Chat:
					client.PeerStorage.AddPeer(c.ID, 0, storage.TypeChat, "")
				}
			}
		case *tg.MessagesDialogsSlice:
			for _, user := range d.Users {
				if u, ok := user.AsNotEmpty(); ok {
					client.PeerStorage.AddPeer(u.ID, u.AccessHash, storage.TypeUser, u.Username)
				}
			}
			for _, chat := range d.Chats {
				switch c := chat.(type) {
				case *tg.Channel:
					client.PeerStorage.AddPeer(c.ID, c.AccessHash, storage.TypeChannel, c.Username)
				case *tg.Chat:
					client.PeerStorage.AddPeer(c.ID, 0, storage.TypeChat, "")
				}
			}
		default:
			fmt.Printf("Unknown dialogs type: %T\n", d)
		}
	}

	return nil
}

func BanUser(ctx *ext.Context, chatID int64, userID int64, duration hdur.Duration) error {
	var untilDate int64
	if !duration.IsZero() {
		untilDate = duration.Add(time.Now()).Unix()
	}
	_, err := ctx.BanChatMember(chatID, userID, int(untilDate))
	if err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}
	return nil
}

func RestrictUser(ctx *ext.Context, chatID int64, userID int64, bannedRights tg.ChatBannedRights) error {
	userPeer := &tg.InputPeerUser{
		UserID: userID,
	}

	chatPeer := functions.GetInputPeerClassFromId(ctx.PeerStorage, chatID)
	if chatPeer == nil {
		return fmt.Errorf("failed to get input peer")
	}

	client := ctx.Raw
	switch c := chatPeer.(type) {
	case *tg.InputPeerChannel:
		_, err := client.ChannelsEditBanned(context.Background(), &tg.ChannelsEditBannedRequest{
			Channel: &tg.InputChannel{
				ChannelID:  c.ChannelID,
				AccessHash: c.AccessHash,
			},
			Participant:  userPeer,
			BannedRights: bannedRights,
		})
		if err != nil {
			return fmt.Errorf("failed to restrict user in channel: %w", err)
		}
	case *tg.InputPeerChat:
		_, err := client.MessagesDeleteChatUser(context.Background(), &tg.MessagesDeleteChatUserRequest{
			ChatID: c.ChatID,
			UserID: &tg.InputUser{
				UserID:     userPeer.UserID,
				AccessHash: userPeer.AccessHash,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to restrict user in chat: %w", err)
		}
	default:
		return fmt.Errorf("unsupported chat type")
	}

	return nil
}

func GetUserFromArgs(ctx *ext.Context, args *command.Arguments) (*types.User, error) {
	var basicUser *types.User
	var err error

	// Check if a user argument was provided
	if args.GetPositionalEntity(0) != "" {
		// Try to resolve user from the provided argument
		basicUser, err = ResolveUser(ctx, args.GetPositionalEntity(0))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user: %w", err)
		}
	} else if args.Reply != nil {
		// Try to get the user from the replied message
		if args.Reply.Message != nil {
			basicUser, err = UserFromMessage(ctx, args.Reply.Message)
			if err != nil {
				return nil, fmt.Errorf("failed to get user from replied message: %w", err)
			}
		} else {
			return nil, fmt.Errorf("could not get replied message")
		}
	} else {
		return nil, fmt.Errorf("please provide a username/ID or reply to a message")
	}

	return basicUser, nil
}

func UnbanUser(ctx *ext.Context, chatID int64, userID int64) error {
	_, err := ctx.UnbanChatMember(chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}
	return nil
}

func GetEffectiveChatID(u *ext.Update) int64 {
	id := u.GetChat().GetID()
	if id == 0 {
		id = u.GetChannel().GetID()
	}
	if id == 0 {
		id = u.GetUserChat().GetID()
	}
	return id
}

func GetMediaFileNameWithId(media tg.MessageMediaClass) (string, error) {
	switch v := media.(type) {
	case *tg.MessageMediaPhoto: // messageMediaPhoto#695150d7
		f, ok := v.Photo.AsNotEmpty()
		if !ok {
			return "", fmt.Errorf("unknown media type")
		}

		return fmt.Sprintf("%d.png", f.ID), nil
	case *tg.MessageMediaDocument: // messageMediaDocument#4cf4d72d
		var (
			attr             tg.DocumentAttributeClass
			ok               bool
			filenameFromAttr *tg.DocumentAttributeFilename
			f                *tg.Document
			filename         = "undefined"
		)

		f, ok = v.Document.AsNotEmpty()
		if !ok {
			return "", fmt.Errorf("unknown media type")
		}

		for _, attr = range f.Attributes {
			filenameFromAttr, ok = attr.(*tg.DocumentAttributeFilename)
			if ok {
				filename = filenameFromAttr.FileName
			}

			videoAttr, ok := attr.(*tg.DocumentAttributeVideo)
			if ok && videoAttr.RoundMessage {
				fmt.Println(videoAttr.String())
				filename = fmt.Sprintf("round%d.mp4", f.ID)
			}

		}

		return fmt.Sprintf("%d-%s", f.ID, filename), nil
	case *tg.MessageMediaStory: // messageMediaStory#68cb6283
		f, ok := v.Story.(*tg.StoryItem)
		if !ok {
			return "", fmt.Errorf("unknown media type")
		}
		return GetMediaFileNameWithId(f.Media)
	}
	return "", fmt.Errorf("unknown media type")
}

func DownloadMessageMedia(ctx *ext.Context, msg *tg.Message) (string, error) {
	media, ok := msg.GetMedia()
	if !ok {
		return "", fmt.Errorf("no media found in the message")
	}

	filename, err := GetMediaFileNameWithId(media)
	if err != nil {
		return "", fmt.Errorf("failed to get media filename: %w", err)
	}

	// Split filename and extension
	extension := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, extension)

	// Create temp file with pattern that puts extension at end
	tmpFilem, err := os.CreateTemp(os.TempDir(), name+"*"+extension)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer tmpFilem.Close()

	_, err = ctx.DownloadMedia(media, ext.DownloadOutputPath(tmpFilem.Name()), nil)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	return tmpFilem.Name(), nil
}
