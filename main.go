package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/watzon/macron/command"
	"github.com/watzon/macron/config"
	"github.com/watzon/macron/modules"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lj "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// Parse command line arguments
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logFilePath := filepath.Join(cfg.SessionDir, fmt.Sprintf("log-%s.txt", time.Now().Format("2006-01-02-15-04-05")))
	fmt.Printf("Logging to %s\n", logFilePath)

	logWriter := zapcore.AddSync(&lj.Logger{
		Filename:   logFilePath,
		MaxBackups: 3,
		MaxSize:    1,
	})
	logCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		logWriter,
		zap.DebugLevel,
	)
	lg := zap.New(logCore)
	defer func() { _ = lg.Sync() }()

	// Create the client
	client, err := gotgproto.NewClient(
		cfg.AppID,
		cfg.AppHash,
		gotgproto.ClientTypePhone(cfg.Phone),
		&gotgproto.ClientOpts{
			Session:        sessionMaker.SqlSession(sqlite.Open(filepath.Join(cfg.SessionDir, "session.db"))),
			Logger:         lg,
			AutoFetchReply: true,
		},
	)
	if err != nil {
		lg.Fatal("Failed to create client", zap.Error(err))
	}

	// Set up command registry with config's command prefix
	registry := command.NewRegistry(cfg.CommandPrefix)

	// Register modules
	registry.AddModule(modules.NewMiscModule())
	registry.AddModule(modules.NewUserModule())

	// Register all modules with the dispatcher
	lg.Info("Registering modules...")
	registry.RegisterAll(client.Dispatcher)

	// Start the client
	lg.Info("Starting client...")
	fmt.Println("Listening for updates. Press Ctrl+C to stop.")
	client.Idle()
}
