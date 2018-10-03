package queue

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
)

type Queue struct {
	topic *pubsub.Topic
}

func NewQueue(projectID, topicID string) (*Queue, error) {
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
	return &Queue{topic}, nil
}

func (q *Queue) PublishImage(name string, img []byte) error {
	ctx := context.Background()
	attrs := map[string]string{"filename": name}
	result := q.topic.Publish(ctx, &pubsub.Message{Data: img, Attributes: attrs})
	serverID, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("Failed to publish: %v", err)
	}
	fmt.Printf("Published img ID=%s", serverID)
	return nil
}

func (q *Queue) Destroy() {
	q.topic.Stop()
}
