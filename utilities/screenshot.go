package utilities

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // Register JPEG format
	_ "image/png"  // Register PNG format
	"math"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/fogleman/gg"
	"github.com/gotd/td/tg"
)

// MessageStyle contains styling information for a message
type MessageStyle struct {
	BackgroundColor color.Color
	TextColor       color.Color
	UsernameColor   color.Color
	Padding         float64
	FontSize        float64
	AvatarSize      float64
	MaxWidth        float64
	MinWidth        float64 // Minimum width for message bubbles
	LineSpacing     float64 // Spacing between lines of text
}

// DefaultMessageStyle returns the default styling for messages
func DefaultMessageStyle() MessageStyle {
	return MessageStyle{
		BackgroundColor: color.RGBA{34, 95, 140, 255},   // Telegram blue
		TextColor:       color.RGBA{255, 255, 255, 255}, // White
		UsernameColor:   color.RGBA{236, 236, 241, 255}, // Light gray
		Padding:         12,                             // Slightly smaller padding
		FontSize:        14,
		AvatarSize:      36,  // Slightly smaller avatar
		MaxWidth:        400, // Narrower messages
		MinWidth:        200, // Minimum bubble width
		LineSpacing:     1.2, // Line spacing multiplier
	}
}

// MessageData contains all the information needed to render a message
type MessageData struct {
	User      *types.User
	Text      string
	Entities  []tg.MessageEntityClass
	Avatar    image.Image
	Timestamp int64 // Unix timestamp
}

// GetUserAvatar fetches and returns a user's profile photo
func GetUserAvatar(ctx *ext.Context, user *types.User) (image.Image, error) {
	// Try to get profile photos directly first
	photos, err := ctx.Raw.PhotosGetUserPhotos(ctx.Context, &tg.PhotosGetUserPhotosRequest{
		UserID: &tg.InputUser{
			UserID:     user.ID,
			AccessHash: user.AccessHash,
		},
		Limit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user photos: %w", err)
	}

	var photo *tg.Photo
	switch p := photos.(type) {
	case *tg.PhotosPhotos:
		if len(p.Photos) > 0 {
			photo = p.Photos[0].(*tg.Photo)
		}
	case *tg.PhotosPhotosSlice:
		if len(p.Photos) > 0 {
			photo = p.Photos[0].(*tg.Photo)
		}
	}

	if photo == nil {
		return nil, fmt.Errorf("no profile photo found")
	}

	// Find the smallest photo size
	var smallestSize *tg.PhotoSize
	for _, size := range photo.Sizes {
		if ps, ok := size.(*tg.PhotoSize); ok {
			if smallestSize == nil || (ps.W < smallestSize.W) {
				smallestSize = ps
			}
		}
	}

	if smallestSize == nil {
		return nil, fmt.Errorf("no suitable photo size found")
	}

	// Download the photo
	fileData, err := ctx.Raw.UploadGetFile(ctx.Context, &tg.UploadGetFileRequest{
		Location: &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     smallestSize.Type,
		},
		Limit: 1024 * 1024, // 1MB limit should be enough for avatar
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download photo: %w", err)
	}

	// Convert the file data to bytes
	var fileBytes []byte
	switch f := fileData.(type) {
	case *tg.UploadFile:
		fileBytes = f.Bytes
	default:
		return nil, fmt.Errorf("unexpected file type: %T", fileData)
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode photo: %w", err)
	}

	return img, nil
}

// ProcessMessageEntities applies formatting to message text based on entities
func ProcessMessageEntities(text string, entities []tg.MessageEntityClass) string {
	if len(entities) == 0 {
		return text
	}

	// Sort entities by offset to process them in order
	// TODO: Implement sorting if needed

	var result string
	lastPos := 0

	for _, entity := range entities {
		switch e := entity.(type) {
		case *tg.MessageEntityBold:
			result += text[lastPos:e.Offset]
			result += fmt.Sprintf("**%s**", text[e.Offset:e.Offset+e.Length])
			lastPos = e.Offset + e.Length
		case *tg.MessageEntityItalic:
			result += text[lastPos:e.Offset]
			result += fmt.Sprintf("_%s_", text[e.Offset:e.Offset+e.Length])
			lastPos = e.Offset + e.Length
		case *tg.MessageEntityCode:
			result += text[lastPos:e.Offset]
			result += fmt.Sprintf("`%s`", text[e.Offset:e.Offset+e.Length])
			lastPos = e.Offset + e.Length
			// Add more entity types as needed
		}
	}

	result += text[lastPos:]
	return result
}

// GenerateMessageScreenshot creates an image containing the messages
func GenerateMessageScreenshot(messages []MessageData, style MessageStyle) (image.Image, error) {
	// Calculate total height needed
	totalHeight := style.Padding // Start with top padding
	for _, msg := range messages {
		height := calculateMessageHeight(msg.Text, style)
		totalHeight += height + style.Padding
	}
	totalHeight += style.Padding // Add bottom padding

	// Create the context with the calculated dimensions
	totalWidth := style.MaxWidth + style.AvatarSize + style.Padding*4 // Add padding for both sides
	dc := gg.NewContext(int(totalWidth), int(totalHeight))

	// Set background color (dark theme)
	dc.SetColor(color.RGBA{17, 27, 33, 255})
	dc.Clear()

	// Load fonts
	if err := dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", style.FontSize); err != nil {
		// Try alternate font paths
		if err := dc.LoadFontFace("/usr/share/fonts/TTF/DejaVuSans.ttf", style.FontSize); err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	// Draw each message
	y := style.Padding * 2 // Start with proper top padding
	for _, msg := range messages {
		messageHeight := calculateMessageHeight(msg.Text, style)

		// Calculate positions
		avatarX := style.Padding * 2                      // Add left padding
		avatarY := y + (messageHeight-style.AvatarSize)/2 // Center avatar vertically
		messageX := style.Padding*3 + style.AvatarSize    // Adjust message position for padding

		// Calculate message width based on content, respecting min and max width
		textWidth := calculateTextWidth(dc, msg.Text, style)
		messageWidth := math.Max(style.MinWidth, math.Min(textWidth+style.Padding*2, style.MaxWidth))

		// Draw message background with rounded corners
		dc.SetColor(style.BackgroundColor)
		dc.DrawRoundedRectangle(messageX, y, messageWidth, messageHeight, 12)
		dc.Fill()

		// Draw avatar if available
		if msg.Avatar != nil {
			// Create a new context for the avatar
			avatarDC := gg.NewContext(int(style.AvatarSize), int(style.AvatarSize))

			// Draw circular mask
			avatarDC.DrawCircle(style.AvatarSize/2, style.AvatarSize/2, style.AvatarSize/2)
			avatarDC.Clip()

			// Scale and draw the avatar
			bounds := msg.Avatar.Bounds()
			scale := style.AvatarSize / math.Min(float64(bounds.Dx()), float64(bounds.Dy()))
			avatarDC.Scale(scale, scale)
			avatarDC.DrawImage(msg.Avatar, 0, 0)

			// Draw the avatar onto the main context
			dc.DrawImage(avatarDC.Image(), int(avatarX), int(avatarY))
		}

		// Draw username with proper font
		dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", style.FontSize)
		dc.SetColor(style.UsernameColor)
		usernameY := y + style.Padding + style.FontSize
		dc.DrawString(FormatUserName(msg.User), messageX+style.Padding, usernameY)

		// Switch back to regular font for message text
		dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", style.FontSize)
		dc.SetColor(style.TextColor)
		textY := usernameY + style.FontSize*style.LineSpacing
		drawWrappedText(dc, msg.Text,
			messageX+style.Padding,
			textY,
			messageWidth-style.Padding*2,
			style.FontSize*style.LineSpacing)

		// Draw timestamp in bottom right
		dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", style.FontSize*0.85)
		dc.SetColor(color.RGBA{180, 180, 180, 255}) // Light gray for timestamp
		timeStr := formatTimestamp(msg.Timestamp)
		timeWidth, _ := dc.MeasureString(timeStr)
		dc.DrawString(timeStr, messageX+messageWidth-timeWidth-style.Padding, y+messageHeight-style.Padding)

		y += messageHeight + style.Padding
	}

	return dc.Image(), nil
}

// calculateMessageHeight estimates the height needed for a message
func calculateMessageHeight(text string, style MessageStyle) float64 {
	// Calculate the height needed for the text
	width := style.MaxWidth - style.Padding*4
	lines := calculateTextLines(text, width, style)

	textHeight := style.FontSize + // Username height
		style.FontSize*style.LineSpacing + // Spacing after username
		float64(len(lines))*style.FontSize*style.LineSpacing // Text lines

	return textHeight + style.Padding*2
}

// calculateTextWidth returns the width needed for the text content
func calculateTextWidth(dc *gg.Context, text string, style MessageStyle) float64 {
	words := splitWords(text)
	maxLineWidth := 0.0

	currentLine := ""
	for _, word := range words {
		test := currentLine
		if currentLine != "" {
			test += " "
		}
		test += word

		w, _ := dc.MeasureString(test)
		if w > style.MaxWidth-style.Padding*4 && currentLine != "" {
			width, _ := dc.MeasureString(currentLine)
			maxLineWidth = math.Max(maxLineWidth, width)
			currentLine = word
		} else {
			currentLine = test
		}
	}

	if currentLine != "" {
		width, _ := dc.MeasureString(currentLine)
		maxLineWidth = math.Max(maxLineWidth, width)
	}

	return maxLineWidth
}

// calculateTextLines returns the number of lines needed for the text
func calculateTextLines(text string, width float64, style MessageStyle) []string {
	words := splitWords(text)
	var lines []string
	currentLine := ""

	for _, word := range words {
		test := currentLine
		if currentLine != "" {
			test += " "
		}
		test += word

		w := style.FontSize * 0.6 * float64(len(test)) // Rough estimate
		if w > width && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = test
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// drawWrappedText draws text with word wrapping
func drawWrappedText(dc *gg.Context, text string, x, y, width, lineHeight float64) {
	words := splitWords(text)
	var line string
	startX := x // Remember the starting X position

	for _, word := range words {
		test := line
		if line != "" {
			test += " "
		}
		test += word

		w, _ := dc.MeasureString(test)
		if w > width && line != "" {
			dc.DrawString(line, startX, y)
			line = word
			y += lineHeight
		} else {
			line = test
		}
	}
	if line != "" {
		dc.DrawString(line, startX, y)
	}
}

// splitWords splits text into words, preserving certain punctuation
func splitWords(text string) []string {
	// This is a basic implementation - can be improved to handle more cases
	return strings.Fields(text)
}

// formatTimestamp converts a Unix timestamp to a human-readable time
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	hour := timestamp % 86400 / 3600
	minute := timestamp % 3600 / 60
	return fmt.Sprintf("%02d:%02d", hour, minute)
}
