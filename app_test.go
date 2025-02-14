package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/chuhlomin/instapaper2rss/pkg/instapaper"
	"github.com/chuhlomin/instapaper2rss/pkg/structs"
)

// Mock for Instapaper interface
type MockInstapaper struct {
	mock.Mock
}

func (m *MockInstapaper) GetBookmarks(params map[string]string) ([]instapaper.Item, error) {
	args := m.Called(params)
	return args.Get(0).([]instapaper.Item), args.Error(1)
}

func (m *MockInstapaper) GetBookmarkText(bookmarkID int) (string, error) {
	args := m.Called(bookmarkID)
	return args.String(0), args.Error(1)
}

// Mock for Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) GetBookmarks() ([]structs.Bookmark, error) {
	args := m.Called()
	return args.Get(0).([]structs.Bookmark), args.Error(1)
}

func (m *MockStorage) WriteBookmark(bookmark *structs.Bookmark) error {
	args := m.Called(bookmark)
	return args.Error(0)
}

// Mock for FeedBuilder interface
type MockFeedBuilder struct {
	mock.Mock
}

func (m *MockFeedBuilder) Build(bookmarks []structs.Bookmark) ([]byte, error) {
	args := m.Called(bookmarks)
	return args.Get(0).([]byte), args.Error(1)
}

func TestApp_Run(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockInstapaper, *MockStorage, *MockFeedBuilder)
		expectedError string
	}{
		{
			name: "successful run, empty storage",
			setupMocks: func(mi *MockInstapaper, ms *MockStorage, mf *MockFeedBuilder) {
				ms.On("GetBookmarks").Return([]structs.Bookmark{}, nil)
				mi.On("GetBookmarks", map[string]string{}).Return([]instapaper.Item{
					{
						Type:     "user",
						UserID:   100,
						Username: "testuser",
					},
					{
						BookmarkID: 1,
						Type:       "bookmark",
						Title:      "Test Bookmark",
						URL:        "https://example.com",
						Hash:       "abc123",
						Time:       1739202544,
					},
				}, nil)
				mi.On("GetBookmarkText", 1).Return("Test content", nil)
				ms.On("WriteBookmark", mock.MatchedBy(func(b *structs.Bookmark) bool {
					return b.ID == 1 && b.Title == "Test Bookmark" && b.Text == "Test content"
				})).Return(nil)
				mf.On("Build", []structs.Bookmark{
					{
						ID:    1,
						Title: "Test Bookmark",
						URL:   "https://example.com",
						Hash:  "abc123",
						Time:  1739202544,
						Text:  "Test content",
					},
				}).Return([]byte("feed"), nil)
			},
			expectedError: "",
		},
		{
			name: "successful run, storage has bookmarks",
			setupMocks: func(mi *MockInstapaper, ms *MockStorage, mf *MockFeedBuilder) {
				ms.On("GetBookmarks").Return([]structs.Bookmark{
					{
						ID:    1,
						Title: "Test Bookmark",
						URL:   "https://example.com",
						Hash:  "abc123",
						Text:  "Test content",
						Time:  1739202544,
					},
				}, nil)
				mi.On("GetBookmarks", map[string]string{"have": "1"}).Return([]instapaper.Item{
					{
						Type:     "user",
						UserID:   100,
						Username: "testuser",
					},
					{
						BookmarkID: 2,
						Type:       "bookmark",
						Title:      "Test Bookmark 2",
						URL:        "https://example.com/2",
						Hash:       "abc456",
						Time:       1739202544,
					},
				}, nil)
				mi.On("GetBookmarkText", 2).Return("Test content 2", nil)
				ms.On("WriteBookmark", mock.MatchedBy(func(b *structs.Bookmark) bool {
					return b.ID == 2 && b.Title == "Test Bookmark 2" && b.Text == "Test content 2"
				})).Return(nil)
				mf.On("Build", []structs.Bookmark{
					{
						ID:    1,
						Title: "Test Bookmark",
						URL:   "https://example.com",
						Hash:  "abc123",
						Time:  1739202544,
						Text:  "Test content",
					},
					{
						ID:    2,
						Title: "Test Bookmark 2",
						URL:   "https://example.com/2",
						Hash:  "abc456",
						Time:  1739202544,
						Text:  "Test content 2",
					},
				}).Return([]byte("feed"), nil)
			},
			expectedError: "",
		},
		{
			name: "GetBookmarks from Storage error",
			setupMocks: func(mi *MockInstapaper, ms *MockStorage, mf *MockFeedBuilder) {
				ms.On("GetBookmarks").Return([]structs.Bookmark{}, fmt.Errorf("storage error"))
			},
			expectedError: "error getting existing bookmarks: storage error",
		},
		{
			name: "GetBookmarks from Instapaper error",
			setupMocks: func(mi *MockInstapaper, ms *MockStorage, mf *MockFeedBuilder) {
				ms.On("GetBookmarks").Return([]structs.Bookmark{}, nil)
				mi.On("GetBookmarks", map[string]string{}).Return([]instapaper.Item{}, fmt.Errorf("API error"))
			},
			expectedError: "error getting bookmarks: API error",
		},
		{
			name: "GetBookmarkText error",
			setupMocks: func(mi *MockInstapaper, ms *MockStorage, mf *MockFeedBuilder) {
				ms.On("GetBookmarks").Return([]structs.Bookmark{}, nil)
				mi.On("GetBookmarks", map[string]string{}).Return([]instapaper.Item{
					{
						BookmarkID: 1,
						Type:       "bookmark",
						Title:      "Test Bookmark",
						URL:        "https://example.com",
						Hash:       "abc123",
						Time:       1739202544,
					},
				}, nil)
				mi.On("GetBookmarkText", 1).Return("", fmt.Errorf("text fetch error"))
			},
			expectedError: "error getting bookmark 1 text: text fetch error",
		},
		{
			name: "WriteBookmark error",
			setupMocks: func(mi *MockInstapaper, ms *MockStorage, mf *MockFeedBuilder) {
				ms.On("GetBookmarks").Return([]structs.Bookmark{}, nil)
				mi.On("GetBookmarks", map[string]string{}).Return([]instapaper.Item{
					{
						BookmarkID: 1,
						Type:       "bookmark",
						Title:      "Test Bookmark",
						URL:        "https://example.com",
						Hash:       "abc123",
						Time:       1739202544,
					},
				}, nil)
				mi.On("GetBookmarkText", 1).Return("Test content", nil)
				ms.On("WriteBookmark", mock.Anything).Return(fmt.Errorf("storage error"))
			},
			expectedError: "error writing bookmark 1 text: storage error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInstapaper := new(MockInstapaper)
			mockStorage := new(MockStorage)
			mockFeedBuilder := new(MockFeedBuilder)

			tt.setupMocks(mockInstapaper, mockStorage, mockFeedBuilder)

			_, err := NewApp(mockInstapaper, mockStorage, mockFeedBuilder).Run("testdata/atom.xml")

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockInstapaper.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}
