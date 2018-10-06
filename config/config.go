package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
)

type WebConfig struct {
	commonConfig
	TopicID string
}

type WorkerConfig struct {
	commonConfig
	SubscriptionID string
	VisionURL      string
	VisionAPIKey   string
}

type commonConfig struct {
	ConnectionString string
	ProjectID        string
	gcpKey           os.File
}

const (
	Port                     = "8080"
	cloudPubSubServiceName   = "cloud-pubsub"
	gcpAppCredentialsEnvName = "GOOGLE_APPLICATION_CREDENTIALS"
	gcpProjectEnvName        = "GOOGLE_CLOUD_PROJECT"
	pubsubSubEnvName         = "PUBSUB_SUBSCRIPTION"
	pubsubTopicEnvName       = "PUBSUB_TOPIC"
)

func NewWebConfig() (WebConfig, error) {
	common, err := newCommonConfig()
	if err != nil {
		return WebConfig{}, fmt.Errorf("could not parse common config env %+v", err)
	}

	topicID, err := parsePubEnv()
	if err != nil {
		return WebConfig{}, fmt.Errorf("could not parse pub env %+v", err)
	}

	return WebConfig{common, topicID}, nil
}

func NewWorkerConfig() (WorkerConfig, error) {
	common, err := newCommonConfig()
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("could not parse common config env %+v", err)
	}

	subscriptionID, err := parseSubEnv()
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("could not parse sub env %+v", err)
	}

	apiKey, url, err := parseVisionEnv()
	if err != nil {
		log.Fatalf("Could not parse vision service env")
	}

	return WorkerConfig{common, subscriptionID, url, apiKey}, nil
}

func (c commonConfig) RemoveTmpFile() {
	os.Remove(c.gcpKey.Name())
}

func newCommonConfig() (commonConfig, error) {
	key, projectID, err := parseKeyAndProjectIDFromEnv()
	if err != nil {
		return commonConfig{}, fmt.Errorf("could not parse pubsub env %+v", err)
	}

	var tmpFile os.File
	if _, ok := os.LookupEnv(gcpAppCredentialsEnvName); !ok {
		tmpFile, err := writeGCPKeyfile(key)
		if err != nil {
			return commonConfig{}, fmt.Errorf("could not write gcp file")
		}

		os.Setenv(gcpAppCredentialsEnvName, tmpFile.Name())
	}

	conn, err := parsePostgresEnv()
	if err != nil {
		return commonConfig{}, fmt.Errorf("could not parse postgres env %+v", err)
	}

	return commonConfig{conn, projectID, tmpFile}, nil
}

func parsePostgresEnv() (conn string, err error) {
	if connectionString, ok := os.LookupEnv("POSTGRESQL_URI"); ok {
		// in k8s
		return connectionString, nil
	}

	service, err := readFirstServiceWithLabel("azure-postgresql-9-6")
	if err != nil {
		return conn, err
	}

	conn, ok := service.CredentialString("uri")

	if !ok {
		return conn, errors.New("could not load postgres uri")
	}
	return conn, err
}

func parseKeyAndProjectIDFromEnv() (key, projectID string, err error) {
	if projectID, ok := os.LookupEnv(gcpProjectEnvName); ok {
		// k8s
		return key, projectID, nil
	}

	service, err := readFirstServiceWithLabel(cloudPubSubServiceName)
	if err != nil {
		return key, projectID, err
	}

	key, ok := service.CredentialString("privateKeyData")
	if !ok {
		return key, projectID, fmt.Errorf("could not load privatekey")
	}

	projectID, ok = service.CredentialString("projectId")
	if !ok {
		return key, projectID, fmt.Errorf("could not load projectId")
	}

	return key, projectID, nil
}

func parsePubEnv() (topicID string, err error) {
	if topicID, ok := os.LookupEnv(pubsubTopicEnvName); ok {
		// k8s
		return topicID, nil
	}

	// CF
	service, err := readFirstServiceWithLabel(cloudPubSubServiceName)
	if err != nil {
		return topicID, err
	}

	topicID, ok := service.CredentialString("topicId")
	if !ok {
		return topicID, errors.New("Could not find topicId")
	}

	return topicID, nil
}

func parseSubEnv() (subscriptionID string, err error) {
	if subscriptionID, ok := os.LookupEnv(pubsubSubEnvName); ok {
		return subscriptionID, nil
	}

	service, err := readFirstServiceWithLabel(cloudPubSubServiceName)
	if err != nil {
		return subscriptionID, err
	}

	subscriptionID, ok := service.CredentialString("subscriptionId")
	if !ok {
		return subscriptionID, errors.New("Could not find subscriptionId")
	}

	return subscriptionID, nil
}

func parseVisionEnv() (apiKey, url string, err error) {
	if apiKey, ok := os.LookupEnv("VISION_APIKEY"); ok {
		// k8s
		if url, ok := os.LookupEnv("VISION_URL"); ok {
			return apiKey, url, nil
		}
	}

	// CF
	service, err := readFirstServiceWithLabel("watson-vision-combined")
	if err != nil {
		return apiKey, url, err
	}

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

func readFirstServiceWithLabel(label string) (service cfenv.Service, err error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return service, err
	}

	services, err := appEnv.Services.WithLabel(label)
	if err != nil {
		return service, err
	}

	if len(services) != 1 {
		return service, fmt.Errorf("Unexpected number of %s services %d", label, len(services))
	}

	return services[0], nil
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
