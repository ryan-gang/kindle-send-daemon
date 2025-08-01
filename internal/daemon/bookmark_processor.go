package daemon

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ryan-gang/kindle-send-daemon/internal/bookmarks"
	"github.com/ryan-gang/kindle-send-daemon/internal/bookmarks/providers"
	"github.com/ryan-gang/kindle-send-daemon/internal/classifier"
	"github.com/ryan-gang/kindle-send-daemon/internal/config"
	"github.com/ryan-gang/kindle-send-daemon/internal/handler"
	"github.com/ryan-gang/kindle-send-daemon/internal/logger"
	"github.com/ryan-gang/kindle-send-daemon/internal/types"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
)

type ProcessedBookmark struct {
	URL       string    `json:"url"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}

type ProcessedState struct {
	Bookmarks []ProcessedBookmark `json:"bookmarks"`
	LastCheck time.Time           `json:"last_check"`
}

// FileProviderAdapter is no longer needed - FileProvider implements bookmarks.Provider directly

type BookmarkProcessor struct {
	statePath string
	state     ProcessedState
	registry  *bookmarks.Registry
	cfg       config.ConfigProvider
	logger    logger.LoggerInterface
}

func NewBookmarkProcessor(cfg config.ConfigProvider, logger logger.LoggerInterface) (*BookmarkProcessor, error) {
	statePath := filepath.Join(filepath.Dir(cfg.GetPidFile()), "processed_bookmarks.json")

	// Create registry and register providers
	registry := bookmarks.NewRegistry()

	// Register file provider
	fileProvider := providers.NewFileProvider()
	registry.Register(fileProvider)

	// Configure file provider if bookmark path is set
	if cfg.GetBookmarkPath() != "" {
		providerConfig := map[string]interface{}{
			"path": cfg.GetBookmarkPath(),
		}
		fileProvider.Configure(providerConfig)
	}

	processor := &BookmarkProcessor{
		statePath: statePath,
		state:     ProcessedState{Bookmarks: make([]ProcessedBookmark, 0)},
		registry:  registry,
		cfg:       cfg,
		logger:    logger,
	}

	processor.loadState()
	return processor, nil
}

func (bp *BookmarkProcessor) ReadBookmarks() ([]string, error) {
	ctx := context.Background()
	providers := bp.registry.GetEnabled()

	if len(providers) == 0 {
		return nil, fmt.Errorf("no enabled bookmark providers")
	}

	var allBookmarks []bookmarks.Bookmark

	// Collect bookmarks from all providers
	for _, provider := range providers {
		bookmarkList, err := provider.GetBookmarks(ctx)
		if err != nil {
			bp.logger.Errorf("Error getting bookmarks from provider %s: %v", provider.Name(), err)
			util.Red.Printf("Error getting bookmarks from provider %s: %v\n", provider.Name(), err)
			continue
		}
		allBookmarks = append(allBookmarks, bookmarkList...)
	}

	// Convert bookmarks to URL strings for backward compatibility
	var urls []string
	for _, bookmark := range allBookmarks {
		urls = append(urls, bookmark.URL)
	}

	// Filter out already processed bookmarks
	newBookmarks := bp.filterNewBookmarks(urls)
	return newBookmarks, nil
}

func (bp *BookmarkProcessor) filterNewBookmarks(bookmarks []string) []string {
	var newBookmarks []string
	processedHashes := make(map[string]bool)

	for _, processed := range bp.state.Bookmarks {
		processedHashes[processed.Hash] = true
	}

	for _, bookmark := range bookmarks {
		hash := bp.hashBookmark(bookmark)
		if !processedHashes[hash] {
			newBookmarks = append(newBookmarks, bookmark)
		}
	}

	return newBookmarks
}

func (bp *BookmarkProcessor) hashBookmark(bookmark string) string {
	hash := md5.Sum([]byte(bookmark))
	return fmt.Sprintf("%x", hash)
}

func (bp *BookmarkProcessor) ProcessBookmarks(bookmarks []string) ([]string, error) {
	if len(bookmarks) == 0 {
		return []string{}, nil
	}

	downloadedRequests, err := bp.downloadBookmarks(bookmarks)
	if err != nil {
		return nil, err
	}

	if err := bp.sendBookmarksViaEmail(downloadedRequests); err != nil {
		return nil, err
	}

	processedBookmarks := bp.updateProcessedState(bookmarks)
	bp.cleanupOldBookmarks()

	if err := bp.saveState(); err != nil {
		util.Red.Printf("Warning: failed to save processed state: %v\n", err)
	}

	return processedBookmarks, nil
}

func (bp *BookmarkProcessor) downloadBookmarks(bookmarks []string) ([]types.Request, error) {
	downloadRequests := classifier.Classify(bookmarks)
	if len(downloadRequests) == 0 {
		bp.logger.Info("No valid bookmarks to process")
		util.Cyan.Println("No valid bookmarks to process")
		return nil, fmt.Errorf("no valid bookmarks to process")
	}

	bp.logger.Infof("Classified %d bookmarks for processing", len(downloadRequests))

	downloadedRequests := handler.Queue(downloadRequests)
	if len(downloadedRequests) == 0 {
		bp.logger.Warn("No bookmarks were successfully downloaded")
		util.Cyan.Println("No bookmarks were successfully downloaded")
		return nil, fmt.Errorf("no bookmarks were successfully downloaded")
	}

	bp.logger.Infof("Successfully downloaded %d bookmarks", len(downloadedRequests))
	return downloadedRequests, nil
}

func (bp *BookmarkProcessor) sendBookmarksViaEmail(downloadedRequests []types.Request) error {
	timeout := bp.cfg.GetCheckInterval() * 60
	if timeout < 60 {
		timeout = config.DefaultTimeout
	}

	bp.logger.Infof("Sending %d bookmarks via email with timeout %d seconds", len(downloadedRequests), timeout)
	handler.Mail(downloadedRequests, timeout)
	return nil
}

func (bp *BookmarkProcessor) updateProcessedState(bookmarks []string) []string {
	var processedBookmarks []string
	now := time.Now()

	for _, bookmark := range bookmarks {
		hash := bp.hashBookmark(bookmark)
		bp.state.Bookmarks = append(bp.state.Bookmarks, ProcessedBookmark{
			URL:       bookmark,
			Hash:      hash,
			Timestamp: now,
		})
		processedBookmarks = append(processedBookmarks, bookmark)
	}

	bp.state.LastCheck = now
	return processedBookmarks
}

func (bp *BookmarkProcessor) cleanupOldBookmarks() {
	if len(bp.state.Bookmarks) > 1000 {
		sort.Slice(bp.state.Bookmarks, func(i, j int) bool {
			return bp.state.Bookmarks[i].Timestamp.After(bp.state.Bookmarks[j].Timestamp)
		})
		bp.state.Bookmarks = bp.state.Bookmarks[:1000]
	}
}

func (bp *BookmarkProcessor) loadState() {
	data, err := os.ReadFile(bp.statePath)
	if err != nil {
		return
	}

	if err := json.Unmarshal(data, &bp.state); err != nil {
		util.Red.Printf("Warning: failed to load processed state: %v\n", err)
		bp.state = ProcessedState{Bookmarks: make([]ProcessedBookmark, 0)}
	}
}

func (bp *BookmarkProcessor) saveState() error {
	data, err := json.MarshalIndent(bp.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bp.statePath, data, 0644)
}
