package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"golang.org/x/net/context"
)

const (
	port               = "8080"
	gcpProjectEnvName  = "GOOGLE_CLOUD_PROJECT"
	pubsubTopicEnvName = "PUBSUB_TOPIC"
)

var topic *pubsub.Topic

func main() {
	ctx := context.Background()

	appEnv, err := cfenv.Current()
	if err != nil {
		log.Fatalf("Couldn't find env %+v", err)
	}

	service, err := appEnv.Services.WithName("pubsub")
	if err != nil {
		log.Fatalf("Couldn't find env %+v", err)
	}

	projectID, ok := service.CredentialString("projectId")
	if !ok {
		log.Fatalf("Couldn't find project id %+v", ok)
	}

	topicId, ok := service.CredentialString("topicId")
	if !ok {
		log.Fatalf("Couldn't find topic id %+v", ok)
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
	log.Println("Created client")

	topic = client.Topic(topicId)

	// The topic existence test requires the binding to have the 'viewer' role.
	ok, err = topic.Exists(ctx)
	if err != nil {
		log.Fatalf("Error finding topic: %v", err)
	}
	if !ok {
		log.Fatalf("Couldn't find topic %v", topic)
	}
	defer topic.Stop()

	http.HandleFunc("/", getHandler)
	http.HandleFunc("/publish", postHandler)

	log.Println("Listening on port:", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	message := r.FormValue("message")
	result := topic.Publish(ctx, &pubsub.Message{Data: []byte(message)})
	serverID, err := result.Get(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Published message ID=%s", serverID)
	http.Redirect(w, r, "/", http.StatusFound)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<!doctype html><form method='POST' action='/publish'>"+
		"<input required name='message' placeholder='Message'>"+
		"<input type='submit' value='Publish'>"+
		"</form>")
}
