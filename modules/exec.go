package modules

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/watzon/macron/command"
)

// ExecModule contains code execution related commands
type ExecModule struct {
	*command.BaseModule
}

// NewExecModule creates a new exec module
func NewExecModule() *ExecModule {
	m := &ExecModule{
		BaseModule: command.NewBaseModule(
			"exec",
			"Code execution commands",
		),
	}

	// Add commands to the module
	m.AddCommand(execGo)

	return m
}

// Load registers all module commands with the dispatcher
func (m *ExecModule) Load(d dispatcher.Dispatcher, prefix string) {
	m.BaseModule.Load(d, prefix)
}

var execGo = command.NewCommand("exec").
	WithUsage("exec <code>").
	WithAliases("eval").
	WithDescription("Execute Go code using yaegi interpreter").
	WithArguments(
		command.ArgumentDefinition{
			Name:        "trunc",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Truncate the output to the max message length of 4096 characters",
		},
		command.ArgumentDefinition{
			Name:        "file",
			Type:        command.TypeBool,
			Kind:        command.KindNamed,
			Required:    false,
			Default:     false,
			Description: "Force sending the output as a file",
		},
	).
	WithHandler(func(ctx *ext.Context, u *ext.Update, args *command.Arguments) error {
		code := args.GetRestString()
		if code == "" {
			return fmt.Errorf("code argument is required")
		}
		fmt.Println(code)

		// Setup stdout and stderr capture
		var stdout, stderr bytes.Buffer

		// Create a new interpreter instance with IO capture
		i := interp.New(interp.Options{
			Stdout: &stdout,
			Stderr: &stderr,
		})

		// Setup timeout channel
		timeoutChan := make(chan bool, 1)
		done := make(chan bool, 1)

		go func() {
			select {
			case <-time.After(30 * time.Second):
				timeoutChan <- true
			case <-done:
				return
			}
		}()
		defer close(done)

		// Create error channel for the evaluation goroutine
		errChan := make(chan error, 1)
		resultChan := make(chan reflect.Value, 1)

		stdlib.Symbols["macron/macron"] = map[string]reflect.Value{
			"Context": reflect.ValueOf(ctx),
			"Update":  reflect.ValueOf(u),
		}

		// Use the standard library
		if err := i.Use(stdlib.Symbols); err != nil {
			return fmt.Errorf("failed to load stdlib: %v", err)
		}

		// Top level imports
		imports := []string{
			`import "fmt"`,
			`import "strings"`,
			`import "time"`,
			`import "math"`,
			`import "encoding/json"`,
			`import "regexp"`,
			`import "sort"`,
			`import "strconv"`,
			`import "macron/macron"`,
		}

		for _, imp := range imports {
			if _, err := i.Eval(imp); err != nil {
				return fmt.Errorf("failed to import %s: %v", imp, err)
			}
		}

		// Split the code into lines and prepare for evaluation
		lines := strings.Split(code, "\n")

		// Start evaluation in a goroutine
		go func() {
			defer func() {
				done <- true
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("panic in evaluation: %v", r)
				}
			}()

			// Evaluate all lines except the last one
			for _, line := range lines[:len(lines)-1] {
				if line == "" {
					continue
				}
				if _, err := i.Eval(line); err != nil {
					errChan <- err
					return
				}
			}

			// Evaluate the last line
			if v, err := i.Eval(lines[len(lines)-1]); err != nil {
				errChan <- err
			} else {
				resultChan <- v
			}
		}()

		// Wait for either completion or timeout
		var v reflect.Value
		select {
		case <-timeoutChan:
			return fmt.Errorf("execution timed out after 30 seconds")
		case err := <-errChan:
			_, replyErr := ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("Error: %v", err)), &ext.ReplyOpts{})
			if replyErr != nil {
				return fmt.Errorf("failed to send error message: %v", replyErr)
			}
			return nil
		case result := <-resultChan:
			v = result
		}

		// Combine outputs
		var parts []string

		// Add return value if present
		if v.IsValid() && !v.IsZero() {
			parts = append(parts, fmt.Sprintf("Return: %v", v.Interface()))
		}

		// Add stdout if not empty
		if stdoutStr := stdout.String(); stdoutStr != "" {
			parts = append(parts, fmt.Sprintf("Stdout:\n%s", stdoutStr))
		}

		// Add stderr if not empty
		if stderrStr := stderr.String(); stderrStr != "" {
			parts = append(parts, fmt.Sprintf("Stderr:\n%s", stderrStr))
		}

		// Combine all parts or use default message
		var output string
		if len(parts) > 0 {
			output = strings.Join(parts, "\n\n")
		} else {
			output = "Code executed successfully with no output"
		}

		// If the message is too long and truncation is enabled, truncate it
		if args.GetBool("trunc") && len(output) > 4087 {
			output = output[:4087] + "..."
		} else if args.GetBool("file") || len(output) > 4096 {
			// Otherwise send the output as a file instead
			f, err := uploader.NewUploader(ctx.Raw).FromBytes(ctx, "output.txt", []byte(output))
			if err != nil {
				return fmt.Errorf("failed to upload output result document: %v", err)
			}
			_, err = ctx.SendMedia(u.EffectiveChat().GetID(), &tg.MessagesSendMediaRequest{
				Media: &tg.InputMediaUploadedDocument{
					MimeType: "text/plain",
					File:     f,
					Attributes: []tg.DocumentAttributeClass{
						&tg.DocumentAttributeFilename{
							FileName: "output.txt",
						},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to send result: %v", err)
			}
			return nil
		}

		msg := fmt.Sprintf("```\n%s\n```", output)
		_, err := ctx.Reply(u, ext.ReplyTextStyledTextArray(parsemode.StylizeText(msg)), &ext.ReplyOpts{})
		if err != nil {
			return fmt.Errorf("failed to send result: %v", err)
		}
		return nil
	})
