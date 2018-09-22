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
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/Samze/services-demo-basel-2018/worker-app/store"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"golang.org/x/net/context"
)

const (
	port              = "8080"
	gcpProjectEnvName = "GOOGLE_CLOUD_PROJECT"
	pubsubSubEnvName  = "PUBSUB_SUBSCRIPTION"
)

type Storer interface {
	AddImage(img []byte) error
}

var processedIDs []string
var processedIDsLock sync.Mutex

func handleListMessages(s Storer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!DOCTYPE html><title>Pubsub example</title>",
			"<h1>Pubsub example</h1>",
			"<p>Received messages:</p>",
			"<ul>")

		processedIDsLock.Lock()
		for _, id := range processedIDs {
			fmt.Fprintln(w, "<li>", html.EscapeString(id))
		}
		processedIDsLock.Unlock()
		fmt.Fprint(w, "<ul>")

	}
}

func processImg(img []byte) []byte {
	return img
}

func main() {
	cctx, cancel := context.WithCancel(context.Background())

	conn, err := parsePostgresEnv()
	if err != nil {
		log.Fatalf("Could not load postgres env %+v", err)
	}

	key, projectID, subID, err := parsePubSubEnv()
	if err != nil {
		log.Printf("Could not load pubsubenv %+v", err)
	}

	tmpFile, err := writeGCPKeyfile(key)
	if err != nil {
		log.Printf("Could not write gcp key %+v", err)
	}
	defer os.Remove(tmpFile.Name())
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpFile.Name())

	store, err := store.NewStore(conn)
	if err != nil {
		log.Fatalf("Could not connect to store %+v", err)
	}

	sub, err := getSubscriber(projectID, subID)
	if err != nil {
		log.Fatalf("Could not get subscription %+v", err)
	}

	go receiveMessages(cctx, sub, store)
	defer cancel()

	http.HandleFunc("/", handleListMessages(store))
	log.Println("Listening on port: ", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func getSubscriber(projectID, subID string) (*pubsub.Subscription, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client: %v", err)
	}

	sub := client.Subscription(subID)
	ok, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error finding subscription %s: %v", subID, err)
	}
	if !ok {
		return nil, fmt.Errorf("Couldn't find subscription %v", subID)
	}

	return sub, nil
}

func receiveMessages(ctx context.Context, sub *pubsub.Subscription, s Storer) {
	err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		fmt.Printf("Got message ID=%s, payload=[%s]", m.ID, m.Data)

		img := processImg(m.Data)

		err := s.AddImage(img)
		if err != nil {
			log.Printf("Could not store text %+v", err)
		}

		processedIDsLock.Lock()
		processedIDs = append(processedIDs, m.ID)
		processedIDsLock.Unlock()

		m.Ack()
	})

	if err != nil {
		log.Fatalf("Failed to receive: %v", err)
	}
}

func writeGCPKeyfile(key string) (*os.File, error) {
	content := []byte(key)
	tmpFile, err := ioutil.TempFile("", "key")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.Write(content); err != nil {
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}
	return tmpFile, nil
}

func parsePubSubEnv() (key, projectID, subID string, err error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return key, projectID, subID, err
	}

	service, err := appEnv.Services.WithName("pubsub")
	if err != nil {
		return key, projectID, subID, err
	}

	projectID, ok := service.CredentialString("projectId")
	if !ok {
		return key, projectID, subID, errors.New("Could not find projectId")
	}

	subID, ok = service.CredentialString("subscriptionId")
	if !ok {
		return key, projectID, subID, errors.New("Could not find subscriptionId")
	}

	key, ok = service.CredentialString("privateKeyData")
	if !ok {
		return key, projectID, subID, errors.New("Could not find privateKeyData")
	}

	return key, projectID, subID, err
}

func parsePostgresEnv() (conn string, err error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return conn, err
	}

	services, err := appEnv.Services.WithTag("PostgreSQL")
	if err != nil {
		return conn, err
	}

	if len(services) > 1 {
		return conn, errors.New("More than one postgres service found")
	}
	service := services[0]

	conn, ok := service.CredentialString("uri")

	if !ok {
		return conn, fmt.Errorf("could not load uri")
	}
	return conn, err
}
