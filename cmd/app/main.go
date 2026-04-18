package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kirill010106/grpc_auth/internal/config"
	"github.com/kirill010106/grpc_auth/internal/storage"
	"github.com/kirill010106/grpc_auth/internal/storage/sqlite"
)

const secretBytesLen = 32

func main() {
	var (
		configPath string
		appName    string
		appSecret  string
	)

	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.StringVar(&appName, "name", "", "app name")
	flag.StringVar(&appSecret, "secret", "", "app secret (optional)")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if configPath == "" {
		fail("config path is required", nil)
	}
	if appName == "" {
		fail("name is required", nil)
	}

	if appSecret == "" {
		var err error
		appSecret, err = generateSecret(secretBytesLen)
		if err != nil {
			fail("failed to generate secret", err)
		}
	}

	cfg := config.MustLoadPath(configPath)

	store, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		fail("failed to init storage", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := store.SaveApp(ctx, appName, appSecret)
	if err != nil {
		if errors.Is(err, storage.ErrAppExists) {
			fail("app already exists", err)
		}
		fail("failed to save app", err)
	}

	fmt.Printf("app_id=%d\napp_secret=%s\n", id, appSecret)
}

func generateSecret(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func fail(msg string, err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		_, _ = fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}
