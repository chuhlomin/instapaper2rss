package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/chuhlomin/instapaper2rss/pkg/atom"
	"github.com/chuhlomin/instapaper2rss/pkg/bolt"
	"github.com/chuhlomin/instapaper2rss/pkg/instapaper"
)

func main() {
	log.Printf("Starting...")

	if err := run(); err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	log.Printf("Done.")
}

func run() error {
	client, err := createInstapaperClient()
	if err != nil {
		return fmt.Errorf("failed to create Instapaper client: %w", err)
	}

	storage, err := bolt.NewStorage(getEnvVar("STORAGE_PATH", "instapaper.db"))
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer storage.Close()

	newBookmarksCount, err := NewApp(client, storage, atom.FeedBuilder{}).
		Run(getEnvVar("FEED_PATH", "feed.xml"))
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	// if run from GitHub Actions, set output variable
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		if err := writeOutput("new_bookmarks_count", strconv.Itoa(newBookmarksCount)); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}

func getEnvVar(key, defaultValue string) string {
	// Try regular environment variable
	if val := os.Getenv(key); val != "" {
		return val
	}

	// Try with INPUT_ prefix (for GitHub Actions)
	if val := os.Getenv("INPUT_" + key); val != "" {
		return val
	}

	return defaultValue
}

func createInstapaperClient() (*instapaper.Client, error) {
	username := flag.String("username", "", "Instapaper username")
	password := flag.String("password", "", "Instapaper password")
	flag.Parse()

	consumerKey := getEnvVar("INSTAPAPER_CONSUMER_KEY", "")
	if consumerKey == "" {
		return nil, fmt.Errorf("INSTAPAPER_CONSUMER_KEY environment variable is not set")
	}

	consumerSecret := getEnvVar("INSTAPAPER_CONSUMER_SECRET", "")
	if consumerSecret == "" {
		return nil, fmt.Errorf("INSTAPAPER_CONSUMER_SECRET environment variable is not set")
	}

	token := getEnvVar("INSTAPAPER_TOKEN", "")
	tokenSecret := getEnvVar("INSTAPAPER_TOKEN_SECRET", "")

	opts := []instapaper.Option{instapaper.WithTimeout(5 * time.Second)}

	if token != "" && tokenSecret != "" {
		opts = append(opts, instapaper.WithToken(token, tokenSecret))
	} else if *username == "" {
		flag.Usage()
		return nil, fmt.Errorf("username is required if token and token secret are not set")
	}

	client, err := instapaper.NewClient(consumerKey, consumerSecret, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating Instapaper client: %w", err)
	}

	if token == "" || tokenSecret == "" {
		token, secret, err := client.GetToken(*username, *password)
		if err != nil {
			return nil, fmt.Errorf("error getting token: %w", err)
		}

		log.Printf("Save the following environment variables:\n")
		log.Printf("INSTAPAPER_TOKEN=%s\n", token)
		log.Printf("INSTAPAPER_TOKEN_SECRET=%s\n", secret)

		instapaper.WithToken(token, secret)(client)
	}

	return client, err
}

func writeOutput(key, value string) error {
	githubOutput := formatOutput(key, value)
	if githubOutput == "" {
		return nil
	}

	path := os.Getenv("GITHUB_OUTPUT")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf(
			"failed to open result file %q: %v. "+
				"If you are using self-hosted runners "+
				"make sure they are updated to version 2.297.0 or greater",
			path,
			err,
		)
	}
	defer f.Close()

	if _, err = f.WriteString(githubOutput); err != nil {
		return fmt.Errorf("failed to write result to file %q: %w", path, err)
	}

	return nil
}

func formatOutput(name, value string) string {
	if value == "" {
		return ""
	}

	// if value contains new line, use multiline format
	if bytes.ContainsRune([]byte(value), '\n') {
		return fmt.Sprintf("%s<<OUTPUT\n%s\nOUTPUT", name, value)
	}

	return fmt.Sprintf("%s=%s", name, value)
}
