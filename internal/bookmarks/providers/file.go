package providers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryan-gang/kindle-send-daemon/internal/bookmarks"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
)

// FileProvider implements the Provider interface for file-based bookmarks
type FileProvider struct {
	path    string
	enabled bool
}

func NewFileProvider() *FileProvider {
	return &FileProvider{
		enabled: false,
	}
}

func (fp *FileProvider) Name() string {
	return "file"
}

func (fp *FileProvider) IsEnabled() bool {
	return fp.enabled && fp.path != ""
}

// Configure configures the file provider with the given settings
func (fp *FileProvider) Configure(config map[string]interface{}) error {
	if path, ok := config["path"].(string); ok {
		fp.path = path
		fp.enabled = true
		return nil
	}
	return fmt.Errorf("file provider requires 'path' setting")
}

// GetBookmarks retrieves bookmarks from the configured file or directory
func (fp *FileProvider) GetBookmarks(ctx context.Context) ([]bookmarks.Bookmark, error) {
	if !fp.IsEnabled() {
		return nil, fmt.Errorf("file provider is not enabled or configured")
	}

	info, err := os.Stat(fp.path)
	if err != nil {
		return nil, fmt.Errorf("bookmark path does not exist: %v", err)
	}

	var allBookmarks []bookmarks.Bookmark

	if info.IsDir() {
		files, err := os.ReadDir(fp.path)
		if err != nil {
			return nil, fmt.Errorf("error reading bookmark directory: %v", err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			filePath := filepath.Join(fp.path, file.Name())
			bookmarkURLs, err := fp.readBookmarkFile(filePath)
			if err != nil {
				util.Red.Printf("Error reading file %s: %v\n", filePath, err)
				continue
			}

			for _, url := range bookmarkURLs {
				allBookmarks = append(allBookmarks, bookmarks.Bookmark{
					URL:       url,
					Title:     "", // File provider doesn't have titles
					Source:    fp.Name(),
					Timestamp: time.Now(),
				})
			}
		}
	} else {
		bookmarkURLs, err := fp.readBookmarkFile(fp.path)
		if err != nil {
			return nil, fmt.Errorf("error reading bookmark file: %v", err)
		}

		for _, url := range bookmarkURLs {
			allBookmarks = append(allBookmarks, bookmarks.Bookmark{
				URL:       url,
				Title:     "", // File provider doesn't have titles
				Source:    fp.Name(),
				Timestamp: time.Now(),
			})
		}
	}

	return allBookmarks, nil
}

func (fp *FileProvider) readBookmarkFile(filePath string) ([]string, error) {
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
