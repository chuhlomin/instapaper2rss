package atom

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/chuhlomin/instapaper2rss/pkg/structs"
)

type Atom struct {
	XMLName xml.Name `xml:"feed"`
	Xmlns   string   `xml:"xmlns,attr"`
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Link    []Link   `xml:"link"`
	Entry   []Entry  `xml:"entry"`
}

type Link struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type Entry struct {
	Title   string  `xml:"title"`
	Link    Link    `xml:"link"`
	ID      string  `xml:"id"`
	Updated string  `xml:"updated"`
	Summary Summary `xml:"summary"`
}

type Summary struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type FeedBuilder struct{}

func (fb FeedBuilder) Build(bookmarks []structs.Bookmark) ([]byte, error) {
	feed := Atom{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   "Instapaper",
		ID:      "https://github.com/chuhlomin/instapaper2rss",
		Updated: time.Now().Format(time.RFC3339),
		Entry:   make([]Entry, len(bookmarks)),
	}

	for i, b := range bookmarks {
		feed.Entry[i] = Entry{
			Title: b.Title,
			Link: Link{
				Href: b.URL,
			},
			ID:      strconv.Itoa(b.ID),
			Updated: time.Unix(b.Time, 0).Format(time.RFC3339),
			Summary: Summary{
				Type: "html",
				Body: b.Text,
			},
		}
	}

	return xml.MarshalIndent(feed, "", "  ")
}
