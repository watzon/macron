package modules

import (
	"fmt"
	"os"
	"syscall"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/watzon/macron/command"
)

// SystemModule contains system-related commands
type SystemModule struct {
	*command.BaseModule
}

// NewSystemModule creates a new system module
func NewSystemModule() *SystemModule {
	m := &SystemModule{
		BaseModule: command.NewBaseModule(
			"system",
			"Provides system-related commands",
		),
	}

	m.AddCommand(kill)

	return m
}

// Load registers all module commands with the dispatcher
func (m *SystemModule) Load(d dispatcher.Dispatcher, prefix string) {
	m.BaseModule.Load(d, prefix)
}

var kill = command.NewCommand("kill").
	WithUsage("kill").
	WithDescription("Kills the current process").
	WithHandler(func(ctx *ext.Context, u *ext.Update, _ *command.Arguments) error {
		fmt.Println("Killing process...")
		syscall.Kill(os.Getpid(), syscall.SIGTERM)
		return nil
	})
