package daemon

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ryan-gang/kindle-send-daemon/internal/classifier"
	"github.com/ryan-gang/kindle-send-daemon/internal/config"
	"github.com/ryan-gang/kindle-send-daemon/internal/handler"
	"github.com/ryan-gang/kindle-send-daemon/internal/logger"
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

type BookmarkProcessor struct {
	statePath string
	state     ProcessedState
}

func NewBookmarkProcessor() *BookmarkProcessor {
	cfg := config.GetInstance()
	statePath := filepath.Join(filepath.Dir(cfg.PidFile), "processed_bookmarks.json")

	processor := &BookmarkProcessor{
		statePath: statePath,
		state:     ProcessedState{Bookmarks: make([]ProcessedBookmark, 0)},
	}

	processor.loadState()
	return processor
}

func (bp *BookmarkProcessor) ReadBookmarks() ([]string, error) {
	cfg := config.GetInstance()
	bookmarkPath := cfg.BookmarkPath

	info, err := os.Stat(bookmarkPath)
	if err != nil {
		return nil, fmt.Errorf("bookmark path does not exist: %v", err)
	}

	var allBookmarks []string

	if info.IsDir() {
		files, err := os.ReadDir(bookmarkPath)
		if err != nil {
			return nil, fmt.Errorf("error reading bookmark directory: %v", err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			filePath := filepath.Join(bookmarkPath, file.Name())
			bookmarks, err := bp.readBookmarkFile(filePath)
			if err != nil {
				util.Red.Printf("Error reading file %s: %v\n", filePath, err)
				continue
			}
			allBookmarks = append(allBookmarks, bookmarks...)
		}
	} else {
		bookmarks, err := bp.readBookmarkFile(bookmarkPath)
		if err != nil {
			return nil, fmt.Errorf("error reading bookmark file: %v", err)
		}
		allBookmarks = bookmarks
	}

	newBookmarks := bp.filterNewBookmarks(allBookmarks)
	return newBookmarks, nil
}

func (bp *BookmarkProcessor) readBookmarkFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var bookmarks []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			bookmarks = append(bookmarks, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return bookmarks, nil
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

	downloadRequests := classifier.Classify(bookmarks)
	if len(downloadRequests) == 0 {
		logger.Info("No valid bookmarks to process")
		util.Cyan.Println("No valid bookmarks to process")
		return []string{}, nil
	}

	logger.Infof("Classified %d bookmarks for processing", len(downloadRequests))

	downloadedRequests := handler.Queue(downloadRequests)
	if len(downloadedRequests) == 0 {
		logger.Warn("No bookmarks were successfully downloaded")
		util.Cyan.Println("No bookmarks were successfully downloaded")
		return []string{}, nil
	}

	logger.Infof("Successfully downloaded %d bookmarks", len(downloadedRequests))

	cfg := config.GetInstance()
	timeout := cfg.CheckInterval * 60
	if timeout < 60 {
		timeout = config.DefaultTimeout
	}

	logger.Infof("Sending %d bookmarks via email with timeout %d seconds", len(downloadedRequests), timeout)
	handler.Mail(downloadedRequests, timeout)

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

	if len(bp.state.Bookmarks) > 1000 {
		sort.Slice(bp.state.Bookmarks, func(i, j int) bool {
			return bp.state.Bookmarks[i].Timestamp.After(bp.state.Bookmarks[j].Timestamp)
		})
		bp.state.Bookmarks = bp.state.Bookmarks[:1000]
	}

	if err := bp.saveState(); err != nil {
		util.Red.Printf("Warning: failed to save processed state: %v\n", err)
	}

	return processedBookmarks, nil
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
