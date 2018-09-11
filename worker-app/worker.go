/**
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"golang.org/x/net/context"
)

const (
	port              = "8080"
	gcpProjectEnvName = "GOOGLE_CLOUD_PROJECT"
	pubsubSubEnvName  = "PUBSUB_SUBSCRIPTION"
)

var (
	messages     []*pubsub.Message
	messagesLock sync.RWMutex
)

func handleListMessages(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<!DOCTYPE html><title>Pubsub example</title>",
		"<h1>Pubsub example</h1>",
		"<p>Received messages:</p>",
		"<ul>")
	messagesLock.RLock()
	defer messagesLock.RUnlock()
	for _, m := range messages {
		fmt.Fprintln(w, "<li>", html.EscapeString(string(m.Data)))
	}
}

func main() {
	cctx, cancel := context.WithCancel(context.Background())
	go receiveMessages(cctx)
	defer cancel()

	http.HandleFunc("/", handleListMessages)
	log.Println("Listening on port: ", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func receiveMessages(ctx context.Context) {
	appEnv, err := cfenv.Current()
	if err != nil {
		log.Fatalf("Couldn't find appenv %+v", err)
	}

	service, err := appEnv.Services.WithName("pubsub")
	if err != nil {
		log.Fatalf("Couldn't find service %+v", err)
	}

	projectID, ok := service.CredentialString("projectId")
	if !ok {
		log.Fatalf("Couldn't find project id %+v", ok)
	}

	subID, ok := service.CredentialString("subscriptionId")
	if !ok {
		log.Fatalf("Couldn't find sub id %+v", ok)
	}

	key, ok := service.CredentialString("privateKeyData")
	if !ok {
		log.Fatalf("Couldn't find key i%+v", ok)
	}

	content := []byte(key)
	tmpfile, err := ioutil.TempFile("", "key")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpfile.Name())

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	sub := client.Subscription(subID)

	ok, err = sub.Exists(ctx)
	if err != nil {
		log.Fatalf("Error finding subscription %s: %v", subID, err)
	}
	if !ok {
		log.Fatalf("Couldn't find subscription %v", subID)
	}

	err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		log.Printf("Got message ID=%s, payload=[%s]", m.ID, m.Data)
		messagesLock.Lock()
		messages = append(messages, m)
		messagesLock.Unlock()
		m.Ack()
	})

	if err != nil {
		log.Fatalf("Failed to receive: %v", err)
	}
}
