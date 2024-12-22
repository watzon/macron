package logger

import (
	"context"
	"fmt"
	"sync"

	"github.com/celestix/gotgproto"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

// Logger represents our custom logging implementation
type Logger struct {
	client     *gotgproto.Client
	ctx        context.Context
	sender     *message.Sender
	logChannel *tg.InputPeerChannel
	mu         sync.Mutex
}

var (
	instance *Logger
	once     sync.Once
)

// Initialize sets up the global logger instance
func Initialize(ctx context.Context, client *gotgproto.Client, logChannel *tg.InputPeerChannel) {
	once.Do(func() {
		instance = &Logger{
			client:     client,
			ctx:        ctx,
			sender:     message.NewSender(client.API()),
			logChannel: logChannel,
		}
	})
}

// Log is the base logging function that logs a message to the configured log channel with no
// additional formatting
func Log(msg string, args ...interface{}) {
	if instance == nil || instance.logChannel == nil {
		return
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	_, err := instance.sender.To(instance.logChannel).Text(instance.ctx, fmt.Sprintf(msg, args...))
	if err != nil {
		return
	}
}

// Error logs an error message to the configured log channel
func Error(format string, args ...interface{}) {
	Log("❌ Error: "+format, args...)
}

// Info logs an informational message to the configured log channel
func Info(format string, args ...interface{}) {
	Log("ℹ️ Info: "+format, args...)
}

// Warning logs a warning message to the configured log channel
func Warning(format string, args ...interface{}) {
	Log("⚠️ Warning: "+format, args...)
}
