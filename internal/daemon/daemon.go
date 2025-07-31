package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nikhil1raghav/kindle-send/internal/config"
	"github.com/nikhil1raghav/kindle-send/internal/logger"
	"github.com/nikhil1raghav/kindle-send/internal/util"
)

type Daemon struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
	processor *BookmarkProcessor
}

func NewDaemon() *Daemon {
	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		ctx:       ctx,
		cancel:    cancel,
		processor: NewBookmarkProcessor(),
	}
}

func (d *Daemon) Start() error {
	cfg := config.GetInstance()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if !cfg.DaemonEnabled {
		return fmt.Errorf("daemon is not enabled in configuration")
	}

	if cfg.BookmarkPath == "" {
		return fmt.Errorf("bookmark path is not configured")
	}

	if d.isRunning() {
		return fmt.Errorf("daemon is already running")
	}

	if err := logger.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
	}
	defer logger.Close()

	if err := d.writePidFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	interval := time.Duration(cfg.CheckInterval) * time.Minute
	d.ticker = time.NewTicker(interval)

	util.GreenBold.Printf("Kindle-send daemon started, checking bookmarks every %d minutes\n", cfg.CheckInterval)
	util.Cyan.Printf("Monitoring bookmark path: %s\n", cfg.BookmarkPath)
	util.Cyan.Printf("PID file: %s\n", cfg.PidFile)
	util.Cyan.Printf("Log file: %s\n", cfg.LogPath)

	logger.Infof("Daemon started with PID %d", os.Getpid())
	logger.Infof("Monitoring bookmark path: %s", cfg.BookmarkPath)
	logger.Infof("Check interval: %d minutes", cfg.CheckInterval)

	d.processBookmarks()

	for {
		select {
		case <-d.ctx.Done():
			logger.Info("Daemon context cancelled")
			util.Cyan.Println("Daemon context cancelled")
			d.cleanup()
			return nil
		case sig := <-sigChan:
			logger.Infof("Received signal: %v", sig)
			util.Cyan.Printf("Received signal: %v\n", sig)
			d.Stop()
			return nil
		case <-d.ticker.C:
			logger.Info("Starting bookmark check cycle")
			util.Cyan.Printf("Checking bookmarks at %s\n", time.Now().Format("2006-01-02 15:04:05"))
			d.processBookmarks()
		}
	}
}

func (d *Daemon) Stop() {
	logger.Info("Stopping daemon...")
	util.Cyan.Println("Stopping daemon...")

	if d.ticker != nil {
		d.ticker.Stop()
	}

	d.cancel()
	d.cleanup()

	logger.Info("Daemon stopped successfully")
	util.Green.Println("Daemon stopped successfully")
}

func (d *Daemon) processBookmarks() {
	if d.processor == nil {
		logger.Error("Bookmark processor not initialized")
		util.Red.Println("Bookmark processor not initialized")
		return
	}

	bookmarks, err := d.processor.ReadBookmarks()
	if err != nil {
		logger.Errorf("Error reading bookmarks: %v", err)
		util.Red.Printf("Error reading bookmarks: %v\n", err)
		return
	}

	if len(bookmarks) == 0 {
		logger.Info("No new bookmarks found")
		util.Cyan.Println("No new bookmarks found")
		return
	}

	logger.Infof("Found %d new bookmarks to process", len(bookmarks))
	util.CyanBold.Printf("Found %d new bookmarks to process\n", len(bookmarks))

	processed, err := d.processor.ProcessBookmarks(bookmarks)
	if err != nil {
		logger.Errorf("Error processing bookmarks: %v", err)
		util.Red.Printf("Error processing bookmarks: %v\n", err)
		return
	}

	if len(processed) > 0 {
		logger.Infof("Successfully processed and sent %d bookmarks", len(processed))
		util.GreenBold.Printf("Successfully processed and sent %d bookmarks\n", len(processed))
	}
}

func (d *Daemon) isRunning() bool {
	cfg := config.GetInstance()
	if cfg.PidFile == "" {
		return false
	}

	pidData, err := os.ReadFile(cfg.PidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (d *Daemon) writePidFile() error {
	cfg := config.GetInstance()
	pid := os.Getpid()
	return os.WriteFile(cfg.PidFile, []byte(strconv.Itoa(pid)), 0644)
}

func (d *Daemon) cleanup() {
	cfg := config.GetInstance()
	if cfg.PidFile != "" {
		os.Remove(cfg.PidFile)
	}
}

func (d *Daemon) Status() error {
	cfg := config.GetInstance()

	if d.isRunning() {
		pidData, _ := os.ReadFile(cfg.PidFile)
		util.Green.Printf("Daemon is running (PID: %s)\n", string(pidData))
		util.Cyan.Printf("Bookmark path: %s\n", cfg.BookmarkPath)
		util.Cyan.Printf("Check interval: %d minutes\n", cfg.CheckInterval)
		return nil
	} else {
		util.Red.Println("Daemon is not running")
		return fmt.Errorf("daemon is not running")
	}
}
