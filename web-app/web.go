package main

import (
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/Samze/services-demo-basel-2018/web-app/store"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"golang.org/x/net/context"
)

type Storer interface {
	GetProcessedText() ([]string, error)
}

const (
	port = "8080"
)

func main() {
	key, projectID, topicID, err := parsePubSubEnv()
	if err != nil {
		log.Fatalf("could not parse pubsub env %+v", err)
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

	conn, err := parsePostgresEnv()
	if err != nil {
		log.Fatalf("could not parse postgres env %+v", err)
	}

	store, err := store.NewStore(conn)
	if err != nil {
		log.Fatalf("Could not connect to store %+v", err)
	}

	s, err := store.GetProcessedText()
	if err != nil {
		log.Fatalf("Could not get text %+v", err)
	}
	fmt.Println(s)

	http.HandleFunc("/", getHandler(store))
	http.HandleFunc("/publish", postHandler(topic))

	fmt.Println("Listening on port:", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
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

func parsePubSubEnv() (key, projectID, topicID string, err error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return key, projectID, topicID, err
	}

	services, err := appEnv.Services.WithLabel("cloud-pubsub")
	if err != nil {
		return key, projectID, topicID, err
	}

	if len(services) > 1 {
		return key, projectID, topicID, errors.New("More than one pubsub service found")
	}
	service := services[0]

	key, ok := service.CredentialString("privateKeyData")
	if !ok {
		return key, projectID, topicID, fmt.Errorf("could not load privatekey")
	}

	projectID, ok = service.CredentialString("projectId")
	if !ok {
		return key, projectID, topicID, fmt.Errorf("could not load projectId")
	}

	topicID, ok = service.CredentialString("topicId")
	if !ok {
		return key, projectID, topicID, fmt.Errorf("could not load topicId")
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
	fmt.Println("Created client")

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
		fmt.Printf("Published message ID=%s", serverID)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func getHandler(t Storer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		texts, err := t.GetProcessedText()
		if err != nil {
		}

		fmt.Fprintf(w, "<!doctype html><form method='POST' action='/publish'>"+
			"<input required name='message' placeholder='Message'>"+
			"<input type='submit' value='Publish'>"+
			"</form>")

		fmt.Fprintln(w, "<ul>")
		for _, text := range texts {
			fmt.Fprintln(w, "<li>", html.EscapeString(string(text)), "</li>")
		}
		fmt.Fprintln(w, "</ul>")
	}
}
