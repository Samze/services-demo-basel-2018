package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/Samze/services-demo-basel-2018/config"
	"github.com/Samze/services-demo-basel-2018/worker-app/classifier"
	"github.com/Samze/services-demo-basel-2018/worker-app/store"
	"golang.org/x/net/context"
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
		fmt.Fprint(w, "<!DOCTYPE html><title>Worker</title>",
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

	c, err := config.NewWorkerConfig()
	if err != nil {
		log.Fatalf("could not load config %+v", err)
	}
	defer c.RemoveTmpFile()

	sub, err := getSubscriber(c.ProjectID, c.SubscriptionID)
	if err != nil {
		log.Fatalf("Could not get subscription %+v", err)
	}

	store, err := store.NewStore(c.ConnectionString)
	if err != nil {
		log.Fatalf("Could not connect to store %+v", err)
	}

	classifier, err := classifier.NewVision(c.VisionURL, c.VisionAPIKey)
	if err != nil {
		log.Fatalf("Could not parse vision service credentials")
	}

	go receiveMessages(cctx, sub, store, classifier)
	defer cancel()

	http.HandleFunc("/", handleListMessages(store))
	log.Println("Listening on port: ", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", config.Port), nil))
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

func writeLog(msg string) {
	logsLock.Lock()
	logs = append(logs, msg)
	logsLock.Unlock()
}
