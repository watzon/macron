package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/functions"
	"github.com/celestix/gotgproto/types"
)

// ArgumentType represents the type of value an argument accepts
type ArgumentType int

const (
	TypeString ArgumentType = iota
	TypeInt
	TypeFloat
	TypeBool
	TypeEntity // For resolving usernames/phone-numbers/ids
	TypeReply  // For accessing replied-to message content
)

// ArgumentKind represents how an argument is specified in the command
type ArgumentKind int

const (
	KindPositional ArgumentKind = iota
	KindNamed
	KindVariadic
)

// ArgumentDefinition defines an expected argument for a command
type ArgumentDefinition struct {
	Name        string       // Name of the argument
	Type        ArgumentType // Type of value the argument accepts
	Kind        ArgumentKind // How the argument is specified
	Required    bool         // Whether the argument is required
	Default     interface{}  // Default value if not provided
	Description string       // Description for help text
}

// ParsedArgument represents a parsed command argument
type ParsedArgument struct {
	Name     string
	Value    interface{}
	RawValue string // Original string value before parsing
}

// ArgumentError represents an error that occurred during argument parsing
type ArgumentError struct {
	Argument string
	Message  string
}

func (e *ArgumentError) Error() string {
	return fmt.Sprintf("argument '%s': %s", e.Argument, e.Message)
}

// Arguments holds all parsed arguments for a command
type Arguments struct {
	Positional []ParsedArgument
	Named      map[string]ParsedArgument
	Variadic   []ParsedArgument
	Raw        string
	Reply      *types.Message // Holds the replied-to message if present
}

// ResolveEntity attempts to resolve a named entity argument to a Telegram user
func (a *Arguments) ResolveEntity(ctx *ext.Context, name string) (*types.User, error) {
	if raw := a.GetEntity(name); raw != "" {
		// Try username resolution first (with or without @ prefix)
		username := strings.TrimPrefix(raw, "@")
		if chat, err := ctx.ResolveUsername(username); err == nil {
			if user, ok := chat.(*types.User); ok {
				return user, nil
			}
		}

		// Try as numeric ID
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			// Get input peer from ID
			peer := functions.GetInputPeerClassFromId(ctx.PeerStorage, id)
			if peer != nil {
				// Try to resolve using the peer
				if chat, err := ctx.ResolveUsername(fmt.Sprint(id)); err == nil {
					if user, ok := chat.(*types.User); ok {
						return user, nil
					}
				}
			}
		}

		return nil, fmt.Errorf("could not resolve entity: %s", raw)
	}
	return nil, fmt.Errorf("entity argument not found: %s", name)
}

// ResolvePositionalEntity attempts to resolve a positional entity argument to a Telegram user
func (a *Arguments) ResolvePositionalEntity(ctx *ext.Context, index int) (*types.User, error) {
	if raw := a.GetPositionalEntity(index); raw != "" {
		// Try username resolution first (with or without @ prefix)
		username := strings.TrimPrefix(raw, "@")
		if chat, err := ctx.ResolveUsername(username); err == nil {
			if user, ok := chat.(*types.User); ok {
				return user, nil
			}
		}

		// Try as numeric ID
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			// Get input peer from ID
			peer := functions.GetInputPeerClassFromId(ctx.PeerStorage, id)
			if peer != nil {
				// Try to resolve using the peer
				if chat, err := ctx.ResolveUsername(fmt.Sprint(id)); err == nil {
					if user, ok := chat.(*types.User); ok {
						return user, nil
					}
				}
			}
		}

		return nil, fmt.Errorf("could not resolve entity: %s", raw)
	}
	return nil, fmt.Errorf("entity argument not found at index: %d", index)
}

// ResolveVariadicEntities attempts to resolve all variadic entity arguments to Telegram users
func (a *Arguments) ResolveVariadicEntities(ctx *ext.Context) ([]*types.User, error) {
	raws := a.GetVariadicEntities()
	if len(raws) == 0 {
		return nil, nil
	}

	users := make([]*types.User, 0, len(raws))
	for _, raw := range raws {
		// Try username resolution first (with or without @ prefix)
		username := strings.TrimPrefix(raw, "@")
		if chat, err := ctx.ResolveUsername(username); err == nil {
			if user, ok := chat.(*types.User); ok {
				users = append(users, user)
				continue
			}
		}

		// Try as numeric ID
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			// Get input peer from ID
			peer := functions.GetInputPeerClassFromId(ctx.PeerStorage, id)
			if peer != nil {
				// Try to resolve using the peer
				if chat, err := ctx.ResolveUsername(fmt.Sprint(id)); err == nil {
					if user, ok := chat.(*types.User); ok {
						users = append(users, user)
						continue
					}
				}
			}
		}

		return nil, fmt.Errorf("could not resolve entity: %s", raw)
	}

	return users, nil
}

// Get returns the value of a named argument
func (a *Arguments) Get(name string) interface{} {
	if arg, ok := a.Named[name]; ok {
		return arg.Value
	}
	return nil
}

// GetString returns the string value of a named argument
func (a *Arguments) GetString(name string) string {
	if v := a.Get(name); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt returns the int value of a named argument
func (a *Arguments) GetInt(name string) int {
	if v := a.Get(name); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

// GetFloat returns the float64 value of a named argument
func (a *Arguments) GetFloat(name string) float64 {
	if v := a.Get(name); v != nil {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// GetBool returns the bool value of a named argument
func (a *Arguments) GetBool(name string) bool {
	if v := a.Get(name); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetEntity returns the raw value of a named entity argument
func (a *Arguments) GetEntity(name string) string {
	if arg, ok := a.Named[name]; ok {
		return arg.RawValue
	}
	return ""
}

// GetPositionalEntity returns the raw value of a positional entity argument
func (a *Arguments) GetPositionalEntity(index int) string {
	if index >= 0 && index < len(a.Positional) {
		return a.Positional[index].RawValue
	}
	return ""
}

// GetVariadicEntities returns the raw values of all variadic entity arguments
func (a *Arguments) GetVariadicEntities() []string {
	values := make([]string, len(a.Variadic))
	for i, arg := range a.Variadic {
		values[i] = arg.RawValue
	}
	return values
}

// GetPositional returns the value of a positional argument by index
func (a *Arguments) GetPositional(index int) interface{} {
	if index >= 0 && index < len(a.Positional) {
		if a.Positional[index].Value != nil {
			return a.Positional[index].Value
		}
	}
	return nil
}

// GetPositionalString returns the string value of a positional argument
func (a *Arguments) GetPositionalString(index int) string {
	if v := a.GetPositional(index); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetPositionalInt returns the int value of a positional argument
func (a *Arguments) GetPositionalInt(index int) int {
	if v := a.GetPositional(index); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

// GetPositionalFloat returns the float64 value of a positional argument
func (a *Arguments) GetPositionalFloat(index int) float64 {
	if v := a.GetPositional(index); v != nil {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// GetPositionalBool returns the bool value of a positional argument
func (a *Arguments) GetPositionalBool(index int) bool {
	if v := a.GetPositional(index); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetVariadic returns all variadic arguments
func (a *Arguments) GetVariadic() []interface{} {
	values := make([]interface{}, len(a.Variadic))
	for i, arg := range a.Variadic {
		values[i] = arg.Value
	}
	return values
}

// parseValue attempts to parse a string value into the specified type
func parseValue(value string, argType ArgumentType) (interface{}, error) {
	switch argType {
	case TypeString:
		return value, nil
	case TypeInt:
		return strconv.Atoi(value)
	case TypeFloat:
		return strconv.ParseFloat(value, 64)
	case TypeBool:
		return strconv.ParseBool(value)
	case TypeEntity:
		// For entities, we store the raw value and resolve it later when we have context
		return value, nil
	case TypeReply:
		// Reply type doesn't use the value parameter, it's handled by storing the reply message in Arguments.Reply
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported argument type")
	}
}

// ParseArguments parses command arguments according to the provided definitions
func ParseArguments(text string, defs []ArgumentDefinition, msg *types.Message) (*Arguments, error) {
	// Split the command text into parts
	parts := strings.Fields(text)

	args := &Arguments{
		Positional: make([]ParsedArgument, 0),
		Named:      make(map[string]ParsedArgument),
		Variadic:   make([]ParsedArgument, 0),
		Raw:        text,
	}

	// Track current position and found named args
	pos := 0
	namedFound := make(map[string]bool)

	// First pass: Handle named arguments
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "-") {
			name := strings.TrimPrefix(part, "-")

			// Find the argument definition
			var def *ArgumentDefinition
			for _, d := range defs {
				if d.Kind == KindNamed && d.Name == name {
					def = &d
					break
				}
			}

			if def == nil {
				return nil, &ArgumentError{name, "unknown named argument"}
			}

			// For boolean flags, presence implies true unless explicitly set
			if def.Type == TypeBool {
				// Check if next part exists and could be a boolean value
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
					value, err := parseValue(parts[i+1], def.Type)
					if err == nil {
						args.Named[name] = ParsedArgument{Name: name, Value: value, RawValue: parts[i+1]}
						i++ // Skip the value
					} else {
						args.Named[name] = ParsedArgument{Name: name, Value: true, RawValue: "true"}
					}
				} else {
					args.Named[name] = ParsedArgument{Name: name, Value: true, RawValue: "true"}
				}
			} else {
				// Non-boolean arguments require a value
				if i+1 >= len(parts) {
					return nil, &ArgumentError{name, "no value provided"}
				}

				// Parse the value
				value, err := parseValue(parts[i+1], def.Type)
				if err != nil {
					return nil, &ArgumentError{name, fmt.Sprintf("invalid value: %v", err)}
				}

				args.Named[name] = ParsedArgument{Name: name, Value: value, RawValue: parts[i+1]}
				i++ // Skip the value
			}
			namedFound[name] = true
			continue
		}

		// Handle positional and variadic args
		for _, def := range defs {
			if def.Kind == KindPositional && pos == 0 {
				value, err := parseValue(part, def.Type)
				if err != nil {
					return nil, &ArgumentError{def.Name, fmt.Sprintf("invalid value: %v", err)}
				}
				args.Positional = append(args.Positional, ParsedArgument{Name: def.Name, Value: value, RawValue: part})
				pos++
				break
			} else if def.Kind == KindVariadic {
				value, err := parseValue(part, def.Type)
				if err != nil {
					return nil, &ArgumentError{def.Name, fmt.Sprintf("invalid value: %v", err)}
				}
				args.Variadic = append(args.Variadic, ParsedArgument{Name: def.Name, Value: value, RawValue: part})
				break
			}
		}
	}

	// Check required arguments and handle reply type
	for _, def := range defs {
		if def.Type == TypeReply {
			if def.Required && args.Reply == nil {
				return nil, &ArgumentError{def.Name, "reply to a message is required"}
			}
			// For reply type, we don't need to parse a value, just check if we have a reply
			if msg != nil && msg.ReplyToMessage != nil {
				args.Reply = msg.ReplyToMessage
			}
		} else if def.Required {
			if def.Kind == KindNamed {
				if !namedFound[def.Name] {
					if def.Default != nil {
						args.Named[def.Name] = ParsedArgument{Name: def.Name, Value: def.Default}
					} else {
						return nil, &ArgumentError{def.Name, "required argument not provided"}
					}
				}
			} else if def.Kind == KindPositional && len(args.Positional) == 0 {
				if def.Default != nil {
					args.Positional = append(args.Positional, ParsedArgument{Name: def.Name, Value: def.Default})
				} else {
					return nil, &ArgumentError{def.Name, "required argument not provided"}
				}
			}
		}
	}

	fmt.Printf("Args: %+v\n", args)
	return args, nil
}
