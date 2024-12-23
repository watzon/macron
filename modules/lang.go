package modules

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode"
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
	WithUsage("translate [-lang <target_language>] <text>").
	WithDescription("Translates text to another language").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "lang",
			Type:        command.TypeString,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     "English",
			Description: "The target language for translation",
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
		text := args.GetRestString()
		if text == "" {
			if u.EffectiveMessage.ReplyToMessage == nil || u.EffectiveMessage.ReplyToMessage.Message == nil {
				return fmt.Errorf("text argument is required or reply to a message")
			}
			text = strings.TrimSpace(u.EffectiveMessage.ReplyToMessage.Text)
		}
		targetLanguage := args.GetString("lang")
		if llmService == nil {
			_, err := ctx.Reply(u, ext.ReplyTextString("LLM service not initialized"), &ext.ReplyOpts{})
			return err
		}
		translatedText, err := llmService.TranslateText(ctx.Context, text, targetLanguage)
		if err != nil {
			_, err := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Error translating text: %v", err)), &ext.ReplyOpts{})
			return err
		}

		// Delete the command message
		chatId := utilities.GetEffectiveChatID(u)
		err = ctx.DeleteMessages(chatId, []int{u.EffectiveMessage.GetID()})

		if args.GetBool("silent") {
			msg := fmt.Sprintf("🌐 *Translation result*\n\n*Input:* `%s`\n\n*Output:* `%s`", text, translatedText)
			logger.Log(msg)
			return err
		} else {
			msg := fmt.Sprintf("|%s|\n\n*%s*", text, strings.TrimSpace(translatedText))
			_, err = ctx.Reply(u, ext.ReplyTextStyledTextArray(parsemode.StylizeText(msg)), &ext.ReplyOpts{})
			return err
		}
	})
