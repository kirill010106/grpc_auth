package suite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	ssov1 "github.com/kirill010106/grpc_auth/gen/go/sso"
	"github.com/kirill010106/grpc_auth/internal/config"
	authgrpc "github.com/kirill010106/grpc_auth/internal/grpc/auth"
	"github.com/kirill010106/grpc_auth/internal/services/auth"
	"github.com/kirill010106/grpc_auth/internal/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
)

type Suite struct {
	*testing.T
	Cfg        *config.Config
	AuthClient ssov1.AuthClient
}

var (
	suiteOnce     sync.Once
	suiteInstance *Suite
	suiteErr      error
)

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()

	suiteOnce.Do(func() {
		suiteInstance, suiteErr = startSuite()
	})

	if suiteErr != nil {
		t.Fatalf("test suite init failed: %v", suiteErr)
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), suiteInstance.Cfg.GRPC.Timeout)

	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	return ctx, &Suite{
		T:          t,
		Cfg:        suiteInstance.Cfg,
		AuthClient: suiteInstance.AuthClient,
	}
}

func configPath() string {
	const key = "CONFIG_PATH"

	if v := os.Getenv(key); v != "" {
		return v
	}

	return filepath.Join(projectRoot(), "config", "config_local.yaml")
}

func projectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func startSuite() (*Suite, error) {
	cfg := config.MustLoadPath(configPath())

	dbPath, err := tempDBPath()
	if err != nil {
		return nil, err
	}
	cfg.StoragePath = dbPath

	if err := runMigrations(dbPath, filepath.Join(projectRoot(), "migrations"), "migrations"); err != nil {
		return nil, err
	}
	if err := runMigrations(dbPath, filepath.Join(projectRoot(), "tests", "migrations"), "test_migrations"); err != nil {
		return nil, err
	}

	log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))

	store, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		return nil, err
	}

	authService := auth.New(log, store, store, store, cfg.TokenTTL)

	grpcServer := grpc.NewServer()
	authgrpc.Register(grpcServer, authService)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	addr := lis.Addr().String()
	if tcpAddr, ok := lis.Addr().(*net.TCPAddr); ok {
		cfg.GRPC.Port = tcpAddr.Port
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cc, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	authClient := ssov1.NewAuthClient(cc)

	return &Suite{
		Cfg:        cfg,
		AuthClient: authClient,
	}, nil
}

func tempDBPath() (string, error) {
	file, err := os.CreateTemp("", "sso-test-*.db")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", err
	}
	return path, nil
}

func runMigrations(dbPath string, migrationsPath string, migrationsTable string) error {
	if migrationsPath == "" {
		return fmt.Errorf("migrations path is empty")
	}
	if migrationsTable == "" {
		return fmt.Errorf("migrations table is empty")
	}

	sourceURL := "file://" + filepath.ToSlash(migrationsPath)
	dbURL := fmt.Sprintf("sqlite3://%s?x-migrations-table=%s", filepath.ToSlash(dbPath), migrationsTable)

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return err
	}

	return nil
}
