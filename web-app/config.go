package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
)

type Config struct {
	ConnectionString string
	ProjectID        string
	TopicID          string
	gcpKey           os.File
}

func NewConfig() (Config, error) {
	key, projectID, topicID, err := parsePubSubEnv()
	if err != nil {
		return Config{}, fmt.Errorf("could not parse pubsub env %+v", err)
	}

	var tmpFile os.File
	if _, ok := os.LookupEnv(googleAppCredentials); !ok {
		tmpFile, err := writeGCPKeyfile(key)
		if err != nil {
			return Config{}, fmt.Errorf("could not write gcp file")
		}

		os.Setenv(googleAppCredentials, tmpFile.Name())
	}

	conn, err := parsePostgresEnv()
	if err != nil {
		return Config{}, fmt.Errorf("could not parse postgres env %+v", err)
	}

	return Config{conn, projectID, topicID, tmpFile}, nil
}

func (c Config) RemoveTmpFile() {
	os.Remove(c.gcpKey.Name())
}

func parsePostgresEnv() (conn string, err error) {
	if connectionString, ok := os.LookupEnv("POSTGRESQL_URI"); ok {
		// in k8s
		return connectionString, nil
	}

	// in CF
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

func parsePubSubEnv() (key, projectID, topicID string, err error) {
	if projectID, ok := os.LookupEnv("GOOGLE_CLOUD_PROJECT"); ok {
		// assume we're in k8s
		if topicID, ok := os.LookupEnv("PUBSUB_TOPIC"); ok {
			return key, projectID, topicID, nil
		}
	}
	// otherwise we're in CF environment
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
