package command

import (
	"testing"
)

func TestParseArguments(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		defs    []ArgumentDefinition
		wantErr bool
		check   func(*testing.T, *Arguments)
	}{
		{
			name: "basic positional",
			text: "arg1 arg2",
			defs: []ArgumentDefinition{
				{Name: "first", Type: TypeString, Kind: KindPositional},
				{Name: "second", Type: TypeString, Kind: KindPositional},
			},
			check: func(t *testing.T, args *Arguments) {
				if len(args.Positional) != 2 {
					t.Errorf("expected 2 positional args, got %d", len(args.Positional))
				}
				if args.GetPositionalString(0) != "arg1" {
					t.Errorf("expected first arg to be 'arg1', got '%s'", args.GetPositionalString(0))
				}
				if args.GetPositionalString(1) != "arg2" {
					t.Errorf("expected second arg to be 'arg2', got '%s'", args.GetPositionalString(1))
				}
			},
		},
		{
			name: "quoted arguments",
			text: `"quoted arg" normal "escaped \"quote\""`,
			defs: []ArgumentDefinition{
				{Name: "first", Type: TypeString, Kind: KindPositional},
				{Name: "second", Type: TypeString, Kind: KindPositional},
				{Name: "third", Type: TypeString, Kind: KindPositional},
			},
			check: func(t *testing.T, args *Arguments) {
				if args.GetPositionalString(0) != "quoted arg" {
					t.Errorf("expected first arg to be 'quoted arg', got '%s'", args.GetPositionalString(0))
				}
				if args.GetPositionalString(1) != "normal" {
					t.Errorf("expected second arg to be 'normal', got '%s'", args.GetPositionalString(1))
				}
				if args.GetPositionalString(2) != `escaped "quote"` {
					t.Errorf("expected third arg to be 'escaped \"quote\"', got '%s'", args.GetPositionalString(2))
				}
			},
		},
		{
			name: "named arguments",
			text: "-flag1 value1 -flag2 value2",
			defs: []ArgumentDefinition{
				{Name: "flag1", Type: TypeString, Kind: KindNamed},
				{Name: "flag2", Type: TypeString, Kind: KindNamed},
			},
			check: func(t *testing.T, args *Arguments) {
				if args.GetString("flag1") != "value1" {
					t.Errorf("expected flag1 to be 'value1', got '%s'", args.GetString("flag1"))
				}
				if args.GetString("flag2") != "value2" {
					t.Errorf("expected flag2 to be 'value2', got '%s'", args.GetString("flag2"))
				}
			},
		},
		{
			name: "boolean flags",
			text: "-bool1 -bool2 true -bool3 false",
			defs: []ArgumentDefinition{
				{Name: "bool1", Type: TypeBool, Kind: KindNamed},
				{Name: "bool2", Type: TypeBool, Kind: KindNamed},
				{Name: "bool3", Type: TypeBool, Kind: KindNamed},
			},
			check: func(t *testing.T, args *Arguments) {
				if !args.GetBool("bool1") {
					t.Error("expected bool1 to be true")
				}
				if !args.GetBool("bool2") {
					t.Error("expected bool2 to be true")
				}
				if args.GetBool("bool3") {
					t.Error("expected bool3 to be false")
				}
			},
		},
		{
			name: "mixed arguments with rest",
			text: "pos1 -flag1 value1 pos2 remaining text",
			defs: []ArgumentDefinition{
				{Name: "first", Type: TypeString, Kind: KindPositional},
				{Name: "flag1", Type: TypeString, Kind: KindNamed},
				{Name: "second", Type: TypeString, Kind: KindPositional},
			},
			check: func(t *testing.T, args *Arguments) {
				if args.GetPositionalString(0) != "pos1" {
					t.Errorf("expected first positional to be 'pos1', got '%s'", args.GetPositionalString(0))
				}
				if args.GetString("flag1") != "value1" {
					t.Errorf("expected flag1 to be 'value1', got '%s'", args.GetString("flag1"))
				}
				if args.GetPositionalString(1) != "pos2" {
					t.Errorf("expected second positional to be 'pos2', got '%s'", args.GetPositionalString(1))
				}
				if args.GetRest() != "remaining text" {
					t.Errorf("expected rest to be 'remaining text', got '%s'", args.GetRest())
				}
			},
		},
		{
			name: "required arguments",
			text: "",
			defs: []ArgumentDefinition{
				{Name: "required", Type: TypeString, Kind: KindPositional, Required: true},
			},
			wantErr: true,
		},
		{
			name: "type conversion",
			text: "-int 42 -float 3.14 -bool true",
			defs: []ArgumentDefinition{
				{Name: "int", Type: TypeInt, Kind: KindNamed},
				{Name: "float", Type: TypeFloat, Kind: KindNamed},
				{Name: "bool", Type: TypeBool, Kind: KindNamed},
			},
			check: func(t *testing.T, args *Arguments) {
				if args.GetInt("int") != 42 {
					t.Errorf("expected int to be 42, got %d", args.GetInt("int"))
				}
				if args.GetFloat("float") != 3.14 {
					t.Errorf("expected float to be 3.14, got %f", args.GetFloat("float"))
				}
				if !args.GetBool("bool") {
					t.Error("expected bool to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := ParseArguments(tt.text, tt.defs, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArguments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.check != nil {
				tt.check(t, args)
			}
		})
	}
}
