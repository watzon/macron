package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/functions"
	"github.com/celestix/gotgproto/types"
	"github.com/watzon/hdur"
)

// ArgumentType represents the type of value an argument accepts
type ArgumentType int

const (
	TypeString ArgumentType = iota
	TypeInt
	TypeFloat
	TypeBool
	TypeEntity   // For resolving usernames/phone-numbers/ids
	TypeReply    // For accessing replied-to message content
	TypeDuration // For parsing human-readable duration strings
)

// ArgumentKind represents how an argument is specified in the command
type ArgumentKind int

const (
	KindPositional ArgumentKind = iota
	KindNamed
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
	Rest       *ParsedArgument // Holds all remaining text after flags and positional args
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

// ResolveRestEntity attempts to resolve the rest entity argument to a Telegram user
func (a *Arguments) ResolveRestEntity(ctx *ext.Context) (*types.User, error) {
	raw := a.GetRest()
	if raw == "" {
		return nil, nil
	}

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

// GetDuration returns the hdur.Duration value of a named argument
func (a *Arguments) GetDuration(name string) hdur.Duration {
	if v := a.Get(name); v != nil {
		if d, ok := v.(hdur.Duration); ok {
			return d
		}
	}
	return hdur.Duration{}
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

// GetPositionalDuration returns the hdur.Duration value of a positional argument
func (a *Arguments) GetPositionalDuration(index int) hdur.Duration {
	if v := a.GetPositional(index); v != nil {
		if d, ok := v.(hdur.Duration); ok {
			return d
		}
	}
	return hdur.Duration{}
}

// GetRest returns any remaining unparsed text
func (a *Arguments) GetRest() string {
	if a.Rest != nil {
		return a.Rest.RawValue
	}
	return ""
}

// GetRestString is an alias for GetRest for backward compatibility
func (a *Arguments) GetRestString() string {
	return a.GetRest()
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
	case TypeDuration:
		d, err := hdur.ParseDuration(value)
		if err != nil {
			return nil, err
		}
		return d, nil
	default:
		return nil, fmt.Errorf("unsupported argument type")
	}
}

// ParseArguments parses command arguments according to the provided definitions
func ParseArguments(text string, defs []ArgumentDefinition, msg *types.Message) (*Arguments, error) {
	args := &Arguments{
		Named:      make(map[string]ParsedArgument),
		Positional: make([]ParsedArgument, 0),
		Raw:        text,
		Reply:      msg,
	}

	// Initialize runes for character-by-character parsing
	runes := []rune(text)
	pos := 0
	length := len(runes)

	// Skip initial whitespace
	for pos < length && runes[pos] == ' ' {
		pos++
	}

	// Count positional definitions
	var positionalCount int
	for _, def := range defs {
		if def.Kind == KindPositional {
			positionalCount++
		}
	}

	var positionalIndex int

	// Parse arguments in a single pass
	for pos < length {
		// Skip whitespace
		for pos < length && runes[pos] == ' ' {
			pos++
		}
		if pos >= length {
			break
		}

		// Check for named argument
		if runes[pos] == '-' && (pos == 0 || runes[pos-1] == ' ') {
			start := pos
			pos++ // Skip dash

			// Read flag name
			var name strings.Builder
			for pos < length && runes[pos] != ' ' && runes[pos] != '\n' && runes[pos] != '=' {
				name.WriteRune(runes[pos])
				pos++
			}

			// Find matching named definition
			var def *ArgumentDefinition
			for _, d := range defs {
				if d.Kind == KindNamed && d.Name == name.String() {
					def = &d
					break
				}
			}

			if def != nil {
				// Skip equals sign or whitespace if present
				if pos < length && (runes[pos] == '=' || runes[pos] == ' ' || runes[pos] == '\n') {
					pos++
				}

				// For boolean flags, always treat as flag without value
				if def.Type == TypeBool {
					args.Named[def.Name] = ParsedArgument{
						Name:     def.Name,
						Value:    true,
						RawValue: "true",
					}
					continue
				}

				// Parse value
				var value string
				if pos < length && runes[pos] == '"' {
					pos++ // Skip opening quote
					var builder strings.Builder
					for pos < length && runes[pos] != '"' {
						if runes[pos] == '\\' && pos+1 < length {
							pos++
						}
						builder.WriteRune(runes[pos])
						pos++
					}
					if pos < length {
						pos++ // Skip closing quote
					}
					value = builder.String()
				} else {
					var builder strings.Builder
					for pos < length && runes[pos] != ' ' {
						builder.WriteRune(runes[pos])
						pos++
					}
					value = builder.String()
				}

				parsedValue, err := parseValue(value, def.Type)
				if err != nil {
					return nil, &ArgumentError{def.Name, err.Error()}
				}
				args.Named[def.Name] = ParsedArgument{
					Name:     def.Name,
					Value:    parsedValue,
					RawValue: value,
				}
				continue
			} else {
				// Not a valid flag, try as positional
				pos = start
			}
		}

		// Try to parse as positional
		if pos < length {
			// If we've found all positional arguments, collect rest
			if positionalIndex >= positionalCount {
				// Skip any leading whitespace
				for pos < length && runes[pos] == ' ' {
					pos++
				}
				if pos < length {
					rest := strings.TrimSpace(string(runes[pos:]))
					if rest != "" {
						args.Rest = &ParsedArgument{
							Name:     "rest",
							Value:    rest,
							RawValue: rest,
						}
					}
				}
				break
			}

			// Find matching positional definition
			var def *ArgumentDefinition
			for _, d := range defs {
				if d.Kind == KindPositional && positionalIndex == len(args.Positional) {
					def = &d
					break
				}
			}

			if def != nil {
				// Parse value
				var value string
				if pos < length && runes[pos] == '"' {
					pos++ // Skip opening quote
					var builder strings.Builder
					for pos < length && runes[pos] != '"' {
						if runes[pos] == '\\' && pos+1 < length {
							pos++
						}
						builder.WriteRune(runes[pos])
						pos++
					}
					if pos < length {
						pos++ // Skip closing quote
					}
					value = builder.String()
				} else {
					var builder strings.Builder
					for pos < length && runes[pos] != ' ' {
						builder.WriteRune(runes[pos])
						pos++
					}
					value = builder.String()
				}

				parsedValue, err := parseValue(value, def.Type)
				if err != nil {
					return nil, &ArgumentError{def.Name, err.Error()}
				}
				args.Positional = append(args.Positional, ParsedArgument{
					Name:     def.Name,
					Value:    parsedValue,
					RawValue: value,
				})
				positionalIndex++
			}
		}
	}

	// Check for required arguments
	for _, def := range defs {
		if def.Required {
			if def.Kind == KindPositional {
				found := false
				for _, arg := range args.Positional {
					if arg.Name == def.Name {
						found = true
						break
					}
				}
				if !found {
					return nil, &ArgumentError{def.Name, "required positional argument missing"}
				}
			} else {
				if _, ok := args.Named[def.Name]; !ok {
					return nil, &ArgumentError{def.Name, "required named argument missing"}
				}
			}
		}
	}

	return args, nil
}
