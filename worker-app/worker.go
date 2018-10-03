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
	"github.com/Samze/services-demo-basel-2018/worker-app/classifier"
	"github.com/Samze/services-demo-basel-2018/worker-app/store"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"golang.org/x/net/context"
)

const (
	port                  = "8080"
	gcpProjectEnvName     = "GOOGLE_CLOUD_PROJECT"
	pubsubSubEnvName      = "PUBSUB_SUBSCRIPTION"
	appCredentialsEnvName = "GOOGLE_APPLICATION_CREDENTIALS"
)

type Storer interface {
	AddImage(img, classifier []byte) error
}

type Vision interface {
	ClassifyImage(name string, img []byte) ([]byte, error)
}

var logs []string
var logsLock sync.Mutex

func handleListMessages(s Storer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!DOCTYPE html><title>Pubsub example</title>",
			"<h1>Worker</h1>",
			"<p>Logs:</p>",
			"<ul>")

		logsLock.Lock()
		for _, line := range logs {
			fmt.Fprintln(w, "<li>", html.EscapeString(line))
		}
		logsLock.Unlock()
		fmt.Fprint(w, "<ul>")
	}
}

func processImg(name string, img []byte, v Vision) []byte {
	clasiffication, err := v.ClassifyImage(name, img)

	if err != nil {
		log.Printf("Could not classify image %s", err)
	}

	return clasiffication
}

func main() {
	cctx, cancel := context.WithCancel(context.Background())

	conn, err := parsePostgresEnv()
	if err != nil {
		log.Fatalf("Could not load postgres env %+v", err)
	}
	store, err := store.NewStore(conn)
	if err != nil {
		log.Fatalf("Could not connect to store %+v", err)
	}

	key, projectID, subID, err := parsePubSubEnv()
	if err != nil {
		log.Printf("Could not load pubsubenv %+v", err)
	}
	if _, ok := os.LookupEnv(appCredentialsEnvName); !ok {
		tmpFile, err := writeGCPKeyfile(key)
		if err != nil {
			log.Printf("Could not write gcp key %+v", err)
		}
		defer os.Remove(tmpFile.Name())
		os.Setenv(appCredentialsEnvName, tmpFile.Name())
	}

	sub, err := getSubscriber(projectID, subID)
	if err != nil {
		log.Fatalf("Could not get subscription %+v", err)
	}

	apiKey, url, err := parseVisionEnv()
	if err != nil {
		log.Fatalf("Could not parse vision service env")
	}

	classifier, err := classifier.NewVision(url, apiKey)
	if err != nil {
		log.Fatalf("Could not parse vision service credentials")
	}

	go receiveMessages(cctx, sub, store, classifier)
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

func receiveMessages(ctx context.Context, sub *pubsub.Subscription, s Storer, v Vision) {
	err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		writeLog(fmt.Sprintf("Recieved msg on queue with ID: %s", m.ID))
		imgName, _ := m.Attributes["filename"]

		writeLog("Classifying image")
		imageClassification := processImg(imgName, m.Data, v)

		writeLog(fmt.Sprintf("Storing img and classification in DB"))
		err := s.AddImage(m.Data, imageClassification)
		if err != nil {
			log.Printf("Could not store text %+v", err)
		}
		m.Ack()

		writeLog(fmt.Sprintf("Finished processing msg with ID: %s", m.ID))
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
	if projectID, ok := os.LookupEnv(gcpProjectEnvName); ok {
		// k8s
		if subID, ok := os.LookupEnv(pubsubSubEnvName); ok {
			return key, projectID, subID, nil
		}
	}

	// CF
	appEnv, err := cfenv.Current()
	if err != nil {
		return key, projectID, subID, err
	}

	services, err := appEnv.Services.WithLabel("cloud-pubsub")
	if err != nil {
		return key, projectID, subID, err
	}

	if len(services) > 1 {
		return key, projectID, subID, errors.New("More than one pubsub service found")
	}
	service := services[0]

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
	if connectionString, ok := os.LookupEnv("POSTGRESQL_URI"); ok {
		// in k8s
		return connectionString, nil
	}
	appEnv, err := cfenv.Current()
	if err != nil {
		return conn, err
	}

	services, err := appEnv.Services.WithLabel("azure-postgresql-9-6")
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

func parseVisionEnv() (apiKey, url string, err error) {
	if apiKey, ok := os.LookupEnv("VISION_APIKEY"); ok {
		// assume we're in k8s
		if url, ok := os.LookupEnv("VISION_URL"); ok {
			return apiKey, url, nil
		}
	}
	// CF
	appEnv, err := cfenv.Current()
	if err != nil {
		return apiKey, url, err
	}

	services, err := appEnv.Services.WithLabel("watson-vision-combined")
	if err != nil {
		return apiKey, url, err
	}

	if len(services) > 1 {
		return apiKey, url, errors.New("More than one vision service found")
	}
	service := services[0]

	apiKey, ok := service.CredentialString("apikey")
	if !ok {
		return apiKey, url, errors.New("Could not find apikey")
	}

	url, ok = service.CredentialString("url")
	if !ok {
		return apiKey, url, errors.New("Could not find url")
	}

	return apiKey, url, nil
}

func writeLog(msg string) {
	logsLock.Lock()
	logs = append(logs, msg)
	logsLock.Unlock()
}
