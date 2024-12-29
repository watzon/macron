package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/watzon/macron/command"
	"github.com/watzon/macron/logger"
	"github.com/watzon/macron/utilities"
)

// UtilitiesModule contains utility commands
type UtilitiesModule struct {
	*command.BaseModule
}

// NewUtilitiesModule creates a new utilities module
func NewUtilitiesModule() *UtilitiesModule {
	m := &UtilitiesModule{
		BaseModule: command.NewBaseModule(
			"utilities",
			"Provides utility commands",
		),
	}

	m.AddCommand(jsonify)
	m.AddCommand(paste)

	return m
}

// Load registers all module commands with the dispatcher
func (m *UtilitiesModule) Load(d dispatcher.Dispatcher, prefix string) {
	m.BaseModule.Load(d, prefix)
}

var jsonify = command.NewCommand("json").
	WithUsage("json").
	WithDescription("Converts a message to JSON").
	WithHandler(func(ctx *ext.Context, u *ext.Update, _ *command.Arguments) error {
		msg := u.EffectiveMessage
		jsonBytes, err := json.MarshalIndent(msg, "", "    ")
		if err != nil {
			return err
		}

		output := string(jsonBytes)

		// Send as file if too long
		if len(output) > 4096 {
			f, err := uploader.NewUploader(ctx.Raw).FromBytes(ctx, "message.json", []byte(output))
			if err != nil {
				return fmt.Errorf("failed to upload JSON document: %v", err)
			}
			_, err = ctx.SendMedia(u.EffectiveChat().GetID(), &tg.MessagesSendMediaRequest{
				Media: &tg.InputMediaUploadedDocument{
					MimeType: "application/json",
					File:     f,
					Attributes: []tg.DocumentAttributeClass{
						&tg.DocumentAttributeFilename{
							FileName: "message.json",
						},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to send JSON: %v", err)
			}
			return nil
		}

		_, err = ctx.Reply(u, ext.ReplyTextString(output), nil)
		return err
	})

var paste = command.NewCommand("paste").
	WithUsage("paste [--cb]").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "cb",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Default:     false,
			Description: "Paste code blocks separately",
		},
		command.ArgumentDefinition{
			Name:        "silent",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Default:     false,
			Description: "Do not send the paste URL in the reply",
		},
	).
	WithDescription("Creates a paste on 0x45.st from the replied message. Use --cb to paste code blocks separately.").
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		replyTo := args.Reply
		ctx.DeleteMessages(u.EffectiveChat().GetID(), []int{u.EffectiveMessage.ID})

		if replyTo == nil {
			return fmt.Errorf("please reply to a message to create a paste")
		}

		// The message that comes before the URL
		messageText := strings.TrimSpace(args.GetRest())

		handleCodeBlocks := args.GetBool("cb")

		if handleCodeBlocks {
			// Find all code blocks in the message
			text := replyTo.Text
			type codeBlock struct {
				language string
				content  string
			}
			codeBlocks := []codeBlock{}
			entities, ok := replyTo.MapEntities()
			if !ok {
				return fmt.Errorf("failed to map entities")
			}

			fmt.Println(entities)

			for _, entity := range entities {
				if entity.TypeName() == "messageEntityPre" {
					entity := entity.(*tg.MessageEntityPre)
					offset := entity.GetOffset()
					length := entity.GetLength()
					language := entity.Language
					if language == "" {
						language = "txt"
					}
					codeBlocks = append(codeBlocks, codeBlock{
						language: language,
						content:  text[offset : offset+length],
					})
				}
			}

			if len(codeBlocks) == 0 {
				return fmt.Errorf("no code blocks found in the message")
			}

			var urls []string
			for i, block := range codeBlocks {
				url, err := createPaste([]byte(block.content), block.language)
				if err != nil {
					return fmt.Errorf("failed to create paste for code block %d: %v", i+1, err)
				}
				urls = append(urls, url)
			}

			// Send all URLs
			if messageText == "" {
				messageText = fmt.Sprintf("Created %d paste(s):\n%s", len(urls), strings.Join(urls, "\n"))
			} else if strings.Contains(messageText, "%s") {
				messageText = fmt.Sprintf(messageText, strings.Join(urls, "\n"))
			}

			if args.GetBool("silent") {
				logger.Log(messageText)
			} else {
				_, err := ctx.Reply(u, ext.ReplyTextString(messageText), &ext.ReplyOpts{
					ReplyToMessageId: replyTo.GetID(),
				})
				return err
			}
		} else if replyTo.Media != nil {
			path, err := utilities.DownloadMessageMedia(ctx, replyTo.Message)
			if err != nil {
				return fmt.Errorf("failed to download media: %v", err)
			}

			defer os.Remove(path)
			// fmt.Println(path)

			bytes, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read media file: %v", err)
			}

			url, err := createPaste(bytes, filepath.Ext(path))
			if err != nil {
				return fmt.Errorf("failed to create paste: %v", err)
			}

			if messageText == "" {
				messageText = fmt.Sprintf("Created paste: %s", url)
			} else if strings.Contains(messageText, "%s") {
				messageText = fmt.Sprintf(messageText, url)
			}

			if args.GetBool("silent") {
				logger.Log(messageText)
			} else {
				_, err = ctx.Reply(u, ext.ReplyTextString(messageText), &ext.ReplyOpts{
					ReplyToMessageId: replyTo.GetID(),
				})
				return err
			}
			return err
		}

		// Create a single paste from the entire message
		url, err := createPaste([]byte(replyTo.Text), "txt")
		if err != nil {
			return err
		}

		if messageText == "" {
			messageText = fmt.Sprintf("Created paste: %s", url)
		} else if strings.Contains(messageText, "%s") {
			messageText = fmt.Sprintf(messageText, url)
		}

		if args.GetBool("silent") {
			logger.Log(messageText)
		} else {
			_, err = ctx.Reply(u, ext.ReplyTextString(messageText), &ext.ReplyOpts{
				ReplyToMessageId: replyTo.GetID(),
			})
		}
		return err
	})

func createPaste(content []byte, extension string) (string, error) {
	// Strip leading dot from extension
	extension = strings.TrimPrefix(extension, ".")

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file field
	part, err := writer.CreateFormFile("file", "paste."+extension)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}

	// Write content to form field
	if _, err := part.Write(content); err != nil {
		return "", fmt.Errorf("failed to write content: %v", err)
	}

	// Close multipart writer
	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", "https://0x45.st/p", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set content type header
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create paste: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create paste: status %d\n%s", resp.StatusCode, body)
	}

	// Read and parse response
	type pasteResponse struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		DeleteURL string `json:"delete_url"`
		ExpiresAt string `json:"expires_at"`
	}

	var response pasteResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Return the URL directly as it's already a full URL
	return response.URL, nil
}
