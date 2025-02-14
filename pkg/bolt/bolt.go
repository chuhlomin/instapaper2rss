package bolt

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	b "github.com/boltdb/bolt"

	"github.com/chuhlomin/instapaper2rss/pkg/structs"
)

type Storage struct {
	db *b.DB
}

const bucketName = "bookmarks"

func NewStorage(path string) (*Storage, error) {
	db, err := b.Open(
		path,
		0600,
		&b.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *b.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return &Storage{db: db}, nil
}

func (s *Storage) GetBookmarks() ([]structs.Bookmark, error) {
	var bookmarks []structs.Bookmark

	err := s.db.View(func(tx *b.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %q not found", bucketName)
		}

		return b.ForEach(func(k, v []byte) error {
			var bookmark structs.Bookmark
			if err := json.Unmarshal(v, &bookmark); err != nil {
				return err
			}

			bookmarks = append(bookmarks, bookmark)
			return nil
		})
	})

	return bookmarks, err
}

func (s *Storage) WriteBookmark(bookmark *structs.Bookmark) error {
	val, err := json.Marshal(bookmark)
	if err != nil {
		return err
	}

	err = s.db.Update(func(tx *b.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %q not found", bucketName)
		}

		key := []byte(strconv.Itoa(bookmark.ID))
		if err := b.Put(key, val); err != nil {
			return err
		}

		return nil
	})
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}
