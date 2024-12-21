package utilities

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
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
	if _, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		if chat, err := ctx.ResolveUsername(identifier); err == nil {
			if user, ok := chat.(*types.User); ok {
				return user, nil
			}
		}
	}

	return nil, fmt.Errorf("could not resolve user: %s", identifier)
}
