package internal

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
)

func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	req.Header.Set("User-Agent", "gator")
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error occured %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error occured %w", err)
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error occured %w", err)
	}

	var rssFeed RSSFeed

	err = xml.Unmarshal(data, &rssFeed)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error occured %w", err)
	}

	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
	for i := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}

	return &rssFeed, nil

}
