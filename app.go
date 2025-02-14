package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/chuhlomin/instapaper2rss/pkg/instapaper"
	"github.com/chuhlomin/instapaper2rss/pkg/structs"
)

type Instapaper interface {
	GetBookmarks(params map[string]string) ([]instapaper.Item, error)
	GetBookmarkText(bookmarkID int) (string, error)
}

type Storage interface {
	GetBookmarks() ([]structs.Bookmark, error)
	WriteBookmark(bookmark *structs.Bookmark) error
}

type FeedBuilder interface {
	Build(bookmarks []structs.Bookmark) ([]byte, error)
}

type App struct {
	instapaper  Instapaper
	storage     Storage
	feedBuilder FeedBuilder
}

func NewApp(
	instapaper Instapaper,
	storage Storage,
	feedBuilder FeedBuilder,
) *App {
	return &App{
		instapaper:  instapaper,
		storage:     storage,
		feedBuilder: feedBuilder,
	}
}

func (a *App) Run(feedPath string) (int, error) {
	existingBookmarks, err := a.storage.GetBookmarks()
	if err != nil {
		return 0, fmt.Errorf("error getting existing bookmarks: %w", err)
	}

	params := map[string]string{}
	if len(existingBookmarks) > 0 {
		params["have"] = concatBookmarksIDs(existingBookmarks)
	}

	items, err := a.instapaper.GetBookmarks(params)
	if err != nil {
		return 0, fmt.Errorf("error getting bookmarks: %w", err)
	}

	var bookmarks []structs.Bookmark
	for _, item := range items {
		switch item.Type {
		case "bookmark":
			bookmarks = append(bookmarks, structs.Bookmark{
				ID:    item.BookmarkID,
				Title: item.Title,
				URL:   item.URL,
				Time:  item.Time,
				Hash:  item.Hash,
			})
		}
	}

	for i, b := range bookmarks {
		text, err := a.instapaper.GetBookmarkText(b.ID)
		if err != nil {
			return 0, fmt.Errorf("error getting bookmark %d text: %w", b.ID, err)
		}

		b.Text = text
		bookmarks[i] = b
		if err := a.storage.WriteBookmark(&b); err != nil {
			return 0, fmt.Errorf("error writing bookmark %d text: %w", b.ID, err)
		}
	}

	if len(bookmarks) == 0 {
		log.Println("No new bookmarks")
		return 0, nil

		// bookmarks = existingBookmarks
		// log.Printf("Using existing %d bookmarks", len(bookmarks))
	}

	newBookmarksCount := len(bookmarks)
	bookmarks = append(existingBookmarks, bookmarks...)

	b, err := a.feedBuilder.Build(bookmarks)
	if err != nil {
		return 0, fmt.Errorf("error building feed: %w", err)
	}

	return newBookmarksCount, saveFeed(b, feedPath)
}

func concatBookmarksIDs(bookmarks []structs.Bookmark) string {
	result := make([]string, len(bookmarks))
	for i, b := range bookmarks {
		result[i] = strconv.Itoa(b.ID)
	}
	return strings.Join(result, ",")
}

func saveFeed(feed []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(xml.Header); err != nil {
		return fmt.Errorf("error writing XML header: %w", err)
	}

	if _, err := file.Write(feed); err != nil {
		return fmt.Errorf("error writing feed: %w", err)
	}

	return nil
}
