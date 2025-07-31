package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ryan-gang/kindle-send-daemon/internal/config"
	"github.com/ryan-gang/kindle-send-daemon/internal/logger"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
)

type Daemon struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
	processor *BookmarkProcessor
	cfg       config.ConfigProvider
	logger    logger.LoggerInterface
}

func NewDaemon(cfg config.ConfigProvider) (*Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create logger instance
	loggerInstance, err := logger.NewLogger(cfg)
	if err != nil {
		cancel() // Ensure cancel is called on error path
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	processor, err := NewBookmarkProcessor(cfg, loggerInstance)
	if err != nil {
		cancel() // Ensure cancel is called on error path
		return nil, fmt.Errorf("failed to create bookmark processor: %w", err)
	}

	return &Daemon{
		ctx:       ctx,
		cancel:    cancel,
		processor: processor,
		cfg:       cfg,
		logger:    loggerInstance,
	}, nil
}

func (d *Daemon) Start() error {
	if err := d.validateConfiguration(); err != nil {
		return err
	}

	if err := d.initializeServices(); err != nil {
		return err
	}
	defer d.logger.Close()

	sigChan := d.setupSignalHandling()
	d.setupTicker()
	d.logStartupInfo()
	d.processBookmarks()

	return d.runEventLoop(sigChan)
}

func (d *Daemon) validateConfiguration() error {
	if d.cfg == nil {
		return fmt.Errorf("configuration not provided")
	}

	if !d.cfg.IsDaemonEnabled() {
		return fmt.Errorf("daemon is not enabled in configuration")
	}

	if d.cfg.GetBookmarkPath() == "" {
		return fmt.Errorf("bookmark path is not configured")
	}

	if d.isRunning() {
		return fmt.Errorf("daemon is already running")
	}

	return nil
}

func (d *Daemon) initializeServices() error {
	if err := d.writePidFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}

	return nil
}

func (d *Daemon) setupSignalHandling() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sigChan
}

func (d *Daemon) setupTicker() {
	interval := time.Duration(d.cfg.GetCheckInterval()) * time.Minute
	d.ticker = time.NewTicker(interval)
}

func (d *Daemon) logStartupInfo() {
	util.GreenBold.Printf("Kindle-send daemon started, checking bookmarks every %d minutes\n", d.cfg.GetCheckInterval())
	util.Cyan.Printf("Monitoring bookmark path: %s\n", d.cfg.GetBookmarkPath())
	util.Cyan.Printf("PID file: %s\n", d.cfg.GetPidFile())
	util.Cyan.Printf("Log file: %s\n", d.cfg.GetLogPath())

	d.logger.Infof("Daemon started with PID %d", os.Getpid())
	d.logger.Infof("Monitoring bookmark path: %s", d.cfg.GetBookmarkPath())
	d.logger.Infof("Check interval: %d minutes", d.cfg.GetCheckInterval())
}

func (d *Daemon) runEventLoop(sigChan chan os.Signal) error {
	for {
		select {
		case <-d.ctx.Done():
			d.logger.Info("Daemon context cancelled")
			util.Cyan.Println("Daemon context cancelled")
			d.cleanup()
			return nil
		case sig := <-sigChan:
			d.logger.Infof("Received signal: %v", sig)
			util.Cyan.Printf("Received signal: %v\n", sig)
			d.Stop()
			return nil
		case <-d.ticker.C:
			d.logger.Info("Starting bookmark check cycle")
			util.Cyan.Printf("Checking bookmarks at %s\n", time.Now().Format("2006-01-02 15:04:05"))
			d.processBookmarks()
		}
	}
}

func (d *Daemon) Stop() {
	d.logger.Info("Stopping daemon...")
	util.Cyan.Println("Stopping daemon...")

	if d.ticker != nil {
		d.ticker.Stop()
	}

	d.cancel()
	d.cleanup()

	d.logger.Info("Daemon stopped successfully")
	util.Green.Println("Daemon stopped successfully")
}

func (d *Daemon) processBookmarks() {
	if d.processor == nil {
		d.logger.Error("Bookmark processor not initialized")
		util.Red.Println("Bookmark processor not initialized")
		return
	}

	bookmarks, err := d.processor.ReadBookmarks()
	if err != nil {
		d.logger.Errorf("Error reading bookmarks: %v", err)
		util.Red.Printf("Error reading bookmarks: %v\n", err)
		return
	}

	if len(bookmarks) == 0 {
		d.logger.Info("No new bookmarks found")
		util.Cyan.Println("No new bookmarks found")
		return
	}

	d.logger.Infof("Found %d new bookmarks to process", len(bookmarks))
	util.CyanBold.Printf("Found %d new bookmarks to process\n", len(bookmarks))

	processed, err := d.processor.ProcessBookmarks(bookmarks)
	if err != nil {
		d.logger.Errorf("Error processing bookmarks: %v", err)
		util.Red.Printf("Error processing bookmarks: %v\n", err)
		return
	}

	if len(processed) > 0 {
		d.logger.Infof("Successfully processed and sent %d bookmarks", len(processed))
		util.GreenBold.Printf("Successfully processed and sent %d bookmarks\n", len(processed))
	}
}

func (d *Daemon) isRunning() bool {
	if d.cfg.GetPidFile() == "" {
		return false
	}

	pidData, err := os.ReadFile(d.cfg.GetPidFile())
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
	pid := os.Getpid()
	return os.WriteFile(d.cfg.GetPidFile(), []byte(strconv.Itoa(pid)), 0644)
}

func (d *Daemon) cleanup() {
	if d.cfg.GetPidFile() != "" {
		os.Remove(d.cfg.GetPidFile())
	}
}

func (d *Daemon) Status() error {
	if d.isRunning() {
		pidData, _ := os.ReadFile(d.cfg.GetPidFile())
		util.Green.Printf("Daemon is running (PID: %s)\n", string(pidData))
		util.Cyan.Printf("Bookmark path: %s\n", d.cfg.GetBookmarkPath())
		util.Cyan.Printf("Check interval: %d minutes\n", d.cfg.GetCheckInterval())
		return nil
	} else {
		util.Red.Println("Daemon is not running")
		return fmt.Errorf("daemon is not running")
	}
}
