package webhook

// webhook related functionality

import (
	"fmt"
	"net/http"
	"crypto/tls"
	"strings"
)

/*
	Here we push the channel contents to the webhook everytime new content is received, set up as to run forever in a similar manner to the listener, but independent of it (rather than embedding the webhook directly inside the listener)
*/
func Push(ch <-chan string, webhook_url string) {
	for {
		// let's not spam the webhook endpoint and only post when there's actual content
		select {
		case payload := <-ch:
			if len(payload) > 0 {
				webhook(payload, webhook_url)
			}
		}
	}
}

// in our case a webhook is nothing but a simple post request with json
func webhook(load string, webhook_url string) {
	payload := strings.NewReader(load)
	fmt.Println(payload)
	
	// guess what, self signed and corporation certs are a pita, so we skip the check, _be careful you trust the dest_ (in our case it's all in local cluster, we good)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	client.Post(webhook_url, "application/json", payload)
	
}
