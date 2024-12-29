package command

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
)

// Module represents a collection of related commands and functionality
type Module interface {
	// Load registers all module commands with the dispatcher
	Load(d dispatcher.Dispatcher, prefix string)
	// Name returns the module's name
	Name() string
	// Description returns the module's description
	Description() string
	// AddCommand adds a command to the module
	AddCommand(cmd *Command)
	// GetCommands returns the module's commands
	GetCommands() []*Command
}

// BaseModule provides a default implementation of Module
type BaseModule struct {
	name        string
	description string
	commands    []*Command
}

// NewBaseModule creates a new base module
func NewBaseModule(name, description string) *BaseModule {
	return &BaseModule{
		name:        name,
		description: description,
		commands:    make([]*Command, 0),
	}
}

// Name returns the module's name
func (m *BaseModule) Name() string {
	return m.name
}

// Description returns the module's description
func (m *BaseModule) Description() string {
	return m.description
}

// AddCommand adds a command to the module
func (m *BaseModule) AddCommand(cmd *Command) {
	m.commands = append(m.commands, cmd)
}

// GetCommands returns the module's commands
func (m *BaseModule) GetCommands() []*Command {
	return m.commands
}

// Load registers all module commands with the dispatcher
func (m *BaseModule) Load(d dispatcher.Dispatcher, prefix string) {
	for _, cmd := range m.commands {
		cmd.Register(d, prefix)
	}
}

// HandlerFunc is the type for command handlers
type HandlerFunc func(ctx *ext.Context, u *ext.Update, args *Arguments) error

// Command represents a base command structure that all commands should embed
type Command struct {
	// Name is the command name without prefix (e.g. "help")
	Name string
	// Aliases are alternative names for the command (e.g. ["h"] for help)
	Aliases []string
	// Usage is a short usage string (e.g. "help [command]")
	Usage string
	// Description is a longer help text explaining what the command does
	Description string
	// Prefix is the command prefix (e.g. "!", "/"). If empty, uses registry's default prefix
	Prefix string
	// Hidden determines if this command should be hidden from help listings
	Hidden bool
	// Outgoing determines if this command responds to outgoing messages
	Outgoing bool
	// Incoming determines if this command responds to incoming messages
	Incoming bool
	// Arguments defines the expected arguments for this command
	Arguments []ArgumentDefinition
	// Handler is the function that handles the command with parsed arguments
	Handler HandlerFunc
}

// NewCommand creates a new command with the given name
func NewCommand(name string) *Command {
	return &Command{
		Name:     name,
		Outgoing: true, // Default to outgoing only
		Incoming: false,
	}
}

// WithUsage sets the command's usage string
func (c *Command) WithUsage(usage string) *Command {
	c.Usage = usage
	return c
}

// WithDescription sets the command's description
func (c *Command) WithDescription(description string) *Command {
	c.Description = description
	return c
}

// WithPrefix sets a custom prefix for the command
func (c *Command) WithPrefix(prefix string) *Command {
	c.Prefix = prefix
	return c
}

// WithHidden sets whether the command should be hidden from help listings
func (c *Command) WithHidden(hidden bool) *Command {
	c.Hidden = hidden
	return c
}

// WithOutgoing sets whether the command responds to outgoing messages
func (c *Command) WithOutgoing(outgoing bool) *Command {
	c.Outgoing = outgoing
	return c
}

// WithIncoming sets whether the command responds to incoming messages
func (c *Command) WithIncoming(incoming bool) *Command {
	c.Incoming = incoming
	return c
}

// WithHandler sets the command's handler function
func (c *Command) WithHandler(handler HandlerFunc) *Command {
	c.Handler = handler
	return c
}

// WithAliases sets alternative names for the command
func (c *Command) WithAliases(aliases ...string) *Command {
	c.Aliases = aliases
	return c
}

// WithArguments sets the command's argument definitions
func (c *Command) WithArguments(args ...ArgumentDefinition) *Command {
	c.Arguments = args
	return c
}

// Register registers the command with the given dispatcher and default prefix
func (c *Command) Register(d dispatcher.Dispatcher, defaultPrefix string) {
	if c.Handler == nil {
		return // Skip registration if no handler is set
	}

	// Use command's prefix if set, otherwise use default
	prefix := c.Prefix
	if prefix == "" {
		prefix = defaultPrefix
	}

	// Create message filter for messages that match the command or its aliases
	createMessageFilter := func(cmdName string) func(m *types.Message) bool {
		return func(m *types.Message) bool {
			// Check if message starts with the command
			if !strings.HasPrefix(m.Text, prefix+cmdName) {
				return false
			}

			// Check if it's an exact match or followed by a space or end of string
			cmdLen := len(prefix + cmdName)
			if len(m.Text) > cmdLen && m.Text[cmdLen] != ' ' {
				return false
			}

			// Check if message direction matches command settings
			if m.Out && !c.Outgoing {
				return false
			}
			if !m.Out && !c.Incoming {
				return false
			}

			return true
		}
	}

	// Create a wrapper handler that parses arguments
	createHandler := func(cmdName string) func(ctx *ext.Context, u *ext.Update) error {
		return func(ctx *ext.Context, u *ext.Update) error {
			// Extract the argument text (everything after the command)
			cmdPrefix := prefix + cmdName
			argText := strings.TrimSpace(strings.TrimPrefix(u.EffectiveMessage.Text, cmdPrefix))

			// Parse arguments with message context
			args, err := ParseArguments(argText, c.Arguments, u.EffectiveMessage)
			if err != nil {
				return fmt.Errorf("invalid arguments: %v", err)
			}

			// Call the original handler with parsed arguments
			return c.Handler(ctx, u, args)
		}
	}

	// Register the main command handler
	d.AddHandler(handlers.NewMessage(createMessageFilter(c.Name), createHandler(c.Name)))

	// Register handlers for all aliases
	for _, alias := range c.Aliases {
		d.AddHandler(handlers.NewMessage(createMessageFilter(alias), createHandler(alias)))
	}
}
