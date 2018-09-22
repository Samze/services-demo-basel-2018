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
	port = "8080"
)

func main() {
	key, projectID, topicID, err := parseVCAPServices()
	if err != nil {
		log.Fatalf("could not parse env %+v", err)
	}

	tmpFile, err := writeGCPKeyfile(key)
	if err != nil {
		log.Fatalf("could not write gcp file")
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpFile.Name())
	defer os.Remove(tmpFile.Name())

	topic, err := setupTopic(projectID, topicID)
	if err != nil {
		log.Fatalf("could not setup topic %+v", err)
	}

	defer topic.Stop()

	http.HandleFunc("/", getHandler)
	http.HandleFunc("/publish", postHandler(topic))

	log.Println("Listening on port:", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func parseVCAPServices() (key, projectID, topicID string, err error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return key, projectID, topicID, err
	}

	service, err := appEnv.Services.WithName("pubsub")
	if err != nil {
		return key, projectID, topicID, err
	}

	key, ok := service.CredentialString("privateKeyData")
	if !ok {
		return key, projectID, topicID, err
	}

	projectID, ok = service.CredentialString("projectId")
	if !ok {
		return key, projectID, topicID, err
	}

	topicID, ok = service.CredentialString("topicId")
	if !ok {
		return key, projectID, topicID, err
	}

	return key, projectID, topicID, nil
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

func setupTopic(projectID, topicID string) (*pubsub.Topic, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client: %v", err)
	}
	log.Println("Created client")

	topic := client.Topic(topicID)

	ok, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error finding topic: %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("Couldn't find topic %v", topic)
	}
	return topic, nil
}

func postHandler(t *pubsub.Topic) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		message := r.FormValue("message")
		result := t.Publish(ctx, &pubsub.Message{Data: []byte(message)})
		serverID, err := result.Get(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Published message ID=%s", serverID)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<!doctype html><form method='POST' action='/publish'>"+
		"<input required name='message' placeholder='Message'>"+
		"<input type='submit' value='Publish'>"+
		"</form>")
}
