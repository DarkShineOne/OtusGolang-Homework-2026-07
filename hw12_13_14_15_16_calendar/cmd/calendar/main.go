package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/app"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/logger"
	internalgrpc "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/server/grpc"
	internalhttp "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/server/http"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	memorystorage "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage/sql"
	"golang.org/x/sync/errgroup"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "configs/config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	if flag.Arg(0) == "version" {
		printVersion()
		return
	}

	if err := run(configFile); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	config, err := NewConfig(configPath)
	if err != nil {
		return err
	}

	logFile, err := openLogFile(config.Logger.File)
	if err != nil {
		return err
	}
	defer logFile.Close()

	logg, err := logger.New(config.Logger.Level, logFile)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	var eventStorage storage.Storage
	var postgres *sqlstorage.Storage
	switch config.Storage.Type {
	case storageTypeMemory:
		eventStorage = memorystorage.New()
	case storageTypeSQL:
		postgres, err = sqlstorage.New(config.Storage.DSN)
		if err != nil {
			return err
		}
		eventStorage = postgres
	default:
		return fmt.Errorf("unsupported storage type %q", config.Storage.Type)
	}

	if postgres != nil {
		connectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = postgres.Connect(connectCtx)
		cancel()
		if err != nil {
			_ = postgres.Close(context.Background())
			return err
		}
		defer func() {
			if err := postgres.Close(context.Background()); err != nil {
				logg.Error("failed to close postgres: " + err.Error())
			}
		}()
	}

	calendar := app.New(logg, eventStorage)
	httpServer := internalhttp.NewServer(logg, calendar, config.HTTP.Host, config.HTTP.Port)
	grpcServer := internalgrpc.NewServer(logg, calendar, config.GRPC.Host, config.GRPC.Port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logg.Info("calendar is running")

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return httpServer.Start(groupCtx)
	})
	group.Go(func() error {
		return grpcServer.Start(groupCtx)
	})
	return group.Wait()
}

func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return file, nil
}
