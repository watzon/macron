package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/gotd/td/tg"
	"github.com/watzon/macron/command"
	"github.com/watzon/macron/config"
	"github.com/watzon/macron/logger"
	"github.com/watzon/macron/modules"
	"github.com/watzon/macron/utilities"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lj "gopkg.in/natefinch/lumberjack.v2"
)

type Args struct {
	FillPeerStorage bool
}

func parseArgs() Args {
	var args Args
	flag.BoolVar(&args.FillPeerStorage, "fill-peer-storage", false, "Fill peer storage with known users and channels")
	flag.Parse()
	return args
}

func main() {
	// Parse command line arguments
	args := parseArgs()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logFilePath := filepath.Join(cfg.SessionDir, "macron.log")
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

	// Initialize our custom logger
	if cfg.LogChannel != 0 {
		channel := client.PeerStorage.GetInputPeerById(cfg.LogChannel)
		typeName := channel.TypeName()
		if channel != nil && typeName != "inputPeerEmpty" {
			logger.Initialize(context.Background(), client, channel.(*tg.InputPeerChannel))
		}
	}

	// Set up command registry with config's command prefix
	registry := command.NewRegistry(cfg.CommandPrefix)

	// Register modules
	registry.AddModule(modules.NewMiscModule())
	registry.AddModule(modules.NewUserModule())
	registry.AddModule(modules.NewExecModule())
	registry.AddModule(modules.NewSystemModule())

	// Register all modules with the dispatcher
	lg.Info("Registering modules...")
	registry.RegisterAll(client.Dispatcher)

	// Start the client
	lg.Info("Starting client...")
	fmt.Println("Listening for updates. Press Ctrl+C to stop.")

	// Log a message to the log channel
	// logger.Info("Macron started")

	// Fill peer storage if requested
	if args.FillPeerStorage {
		logger.Info("Filling peer storage from dialogs...")
		if err := utilities.FillPeerStorage(client, 250); err == nil {
			logger.Info("Peer storage filled successfully")
		} else {
			logger.Error("Failed to fill peer storage: %v", err)
		}
	}

	client.Idle()
}
