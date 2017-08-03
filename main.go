package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"

	gmail "google.golang.org/api/gmail/v1"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

var progress bool
var byDomain bool
var verbose bool

func init() {
	flag.BoolVar(&progress, "progress", true, "display dot for each email")
	flag.BoolVar(&byDomain, "domains", false, "also display stats about domains")
	flag.BoolVar(&verbose, "verbose", true, "display headings and totals")
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
	totalCounted := 0
	senders := map[string]int{}
	domains := map[string]int{}
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
						totalCounted++
						senders[h.Value]++

						if byDomain {
							domain := h.Value[strings.LastIndex(h.Value, "@")+1 : len(h.Value)-1]
							domains[domain]++
						}

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
		if verbose {
			fmt.Printf("Top senders (in %d emails):\n", total)
		}

		sorted := rank(senders)
		for _, sender := range sorted {
			fmt.Printf("%d\t%s\n", sender.Count, sender.Name)
		}

		if verbose {
			fmt.Printf("%d emails counted\n", totalCounted)
		}

		if byDomain {
			if verbose {
				fmt.Printf("Top domains (in %d emails):\n", total)
			}

			sorted := rank(domains)
			for _, d := range sorted {
				fmt.Printf("%d\t%s\n", d.Count, d.Name)
			}
		}
	}
}
