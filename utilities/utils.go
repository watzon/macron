package utilities

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/storage"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
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
