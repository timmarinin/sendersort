package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	gmail "google.golang.org/api/gmail/v1"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

var progress bool

func init() {
	flag.BoolVar(&progress, "progress", true, "display dot for each email")
}

func main() {
	flag.Parse()
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client: %v", err)
	}

	total := 0
	senders := map[string]int{}
	pageToken := ""

	for {
		req := srv.Users.Messages.List("me").Q("is:inbox")
		if pageToken != "" {
			req.PageToken(pageToken)
		}

		r, err := req.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve messages. %v", err)
		}

		var wg sync.WaitGroup
		var mut sync.Mutex

		for _, m := range r.Messages {
			wg.Add(1)
			go func(m *gmail.Message) {
				defer wg.Done()
				msg, err := srv.Users.Messages.Get("me", m.Id).Format("metadata").Do()
				if progress {
					fmt.Print(".")
				}
				if err != nil {
					log.Fatalf("Could not fetch email with id %s: %v", m.Id, err)
				}

				total++

				for _, h := range msg.Payload.Headers {
					if h.Name == "From" {
						mut.Lock()
						senders[h.Value]++
						mut.Unlock()
						break
					}
				}
			}(m)
		}
		wg.Wait()
		if progress {
			log.Print("\n")
		}
		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
	}

	if total > 0 {
		fmt.Printf("Top senders (in %d emails):\n", total)
		sorted := rank(senders)
		for _, sender := range sorted {
			fmt.Printf("%d\t%s\n", sender.Count, sender.Name)
		}
	}
}
