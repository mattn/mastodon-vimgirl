package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-docomo"
	"github.com/mattn/go-mastodon"
	"golang.org/x/net/html"
)

type config struct {
	Apikey   string          `json:"apikey"`
	VimGirl  docomo.User     `json:"vimgirl"`
	Mastodon mastodon.Config `json:"mastodon"`
}

func textContent(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	var buf bytes.Buffer

	var extractText func(node *html.Node, w *bytes.Buffer)
	extractText = func(node *html.Node, w *bytes.Buffer) {
		if node.Type == html.TextNode {
			data := strings.Trim(node.Data, "\r\n")
			if data != "" {
				w.WriteString(data)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractText(c, w)
		}
		if node.Type == html.ElementNode {
			name := strings.ToLower(node.Data)
			if name == "br" {
				w.WriteString("\n")
			}
		}
	}
	extractText(doc, &buf)
	return buf.String()
}

func main() {
	f, err := os.Open("vimgirl-config.json")
	if err != nil {
		log.Fatal(err)
	}
	var cfg config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	c := docomo.NewClient(cfg.Apikey, cfg.VimGirl)
	m := mastodon.NewClient(&cfg.Mastodon)

	q, err := m.StreamingUser(context.Background(), "vimgirl")
	if err != nil {
		log.Fatal(err)
	}
	for e := range q {
		switch t := e.(type) {
		case *mastodon.NotificationEvent:
			n := t.Notification
			if n.Type != "mention" {
				continue
			}
			text := textContent(n.Status.Content)
			log.Println(n.Status.Account.Acct + ": " + text)
			time.AfterFunc(3*time.Second, func() {
				ret, err := c.Dialogue(text)
				if err != nil {
					log.Println(err)
					return
				}
				text := "@" + n.Status.Account.Acct + " " + ret.Utt
				_, err = m.PostStatus(context.Background(), &mastodon.Toot{
					Status:      text,
					InReplyToID: n.Status.ID,
					Visibility:  "unlisted",
				})
				if err != nil {
					log.Println(err)
				} else {
					log.Println(text)
				}
			})
		}
	}
}
