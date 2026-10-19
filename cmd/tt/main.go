package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"typing_tracker/internal/config"
	"typing_tracker/internal/report"
	"typing_tracker/internal/server"
	"typing_tracker/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve()
	case "today":
		err = printRange("today")
	case "week":
		err = printRange("week")
	case "status":
		err = status()
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tt: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `tt — local coding time for Vim and Emacs

Usage:
  tt serve     listen on 127.0.0.1 for heartbeats
  tt today     time tracked today
  tt week      time tracked across the last 7 days
  tt status    server status and the latest heartbeat

Heartbeats stay in ~/.local/share/typing-tracker/. Nothing leaves this machine.
`)
}

func serve() error {
	cfg, err := config.Load(true)
	if err != nil {
		return err
	}
	if err := server.ValidateBind(cfg.Bind); err != nil {
		return err
	}
	token, err := config.Token(cfg.Dir)
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	db, err := store.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	addr := net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w (is tt serve already running?)", addr, err)
	}
	defer ln.Close()

	api := &server.Server{Token: token, Store: db, Ignore: cfg.Ignore}
	httpServer := &http.Server{
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	fmt.Printf("typing-tracker listening on %s\n", addr)
	fmt.Printf("database %s\n", cfg.DBPath())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		err := httpServer.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
		return <-errCh
	case err := <-errCh:
		return err
	}
}

func printRange(which string) error {
	cfg, err := config.Load(false)
	if err != nil {
		return err
	}
	beats, err := loadBeats(cfg)
	if err != nil {
		return err
	}
	now := time.Now()
	gap := time.Duration(cfg.GapSeconds) * time.Second
	switch which {
	case "today":
		fmt.Print(report.Today(beats, now, gap))
	default:
		fmt.Print(report.Week(beats, now, gap))
	}
	return nil
}

func loadBeats(cfg config.Config) ([]store.Heartbeat, error) {
	if _, err := os.Stat(cfg.DBPath()); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	db, err := store.Open(cfg.DBPath())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -6)
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	return db.Between(start, end)
}

func status() error {
	cfg, err := config.Load(false)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port))
	up := serverUp(addr)
	var last store.Heartbeat
	has := false
	if _, err := os.Stat(cfg.DBPath()); err == nil {
		db, err := store.Open(cfg.DBPath())
		if err != nil {
			return err
		}
		defer db.Close()
		last, has, err = db.Latest()
		if err != nil {
			return err
		}
	}
	fmt.Print(report.Status(up, addr, last, has))
	return nil
}

func serverUp(addr string) bool {
	client := http.Client{Timeout: 400 * time.Millisecond}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
