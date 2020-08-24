package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/stretchr/testify/assert"
)

const (
	container01 = "go-elasticsearch_elasticsearch01_1"
	container02 = "go-elasticsearch_elasticsearch02_1"
	container03 = "go-elasticsearch_elasticsearch03_1"
	docker      = "docker"
	index       = "index"
	start       = "start"
	stop        = "stop"
	sudo        = "su-exec"
	user        = "root"
)

var connection = Connection{}

func TestConnect(t *testing.T) {
	connection.Connect(getDsn())

	reCreateIndex(t)
	insert(t)
	search(t)
}

func TestDisconnect(t *testing.T) {
	connection.Connect(getDsn())
	connection.Disconnect()

	assert.Nil(t, nil)
}

func TestIsConnected(t *testing.T) {
	connection.Connect(getDsn())
	assert.True(t, connection.IsConnected())

	_ = getCmd01Context(stop).Run()
	_ = getCmd02Context(stop).Run()
	_ = getCmd03Context(stop).Run()
	assert.False(t, connection.IsConnected())

	_ = getCmd01Context(start).Run()
	_ = getCmd02Context(start).Run()
	_ = getCmd03Context(start).Run()

	for {
		canBreak := true

		for _, value := range getPartialDsn() {
			if _, err := http.Get(value); err != nil {
				canBreak = false
			}
		}

		if canBreak {
			break
		}

		time.Sleep(time.Second)
	}

	assert.True(t, connection.IsConnected())
}

func TestConnectError(t *testing.T) {
	_ = getCmd01Context(stop).Run()
	_ = getCmd02Context(stop).Run()
	_ = getCmd03Context(stop).Run()

	go func() {
		for range time.After(time.Second) {
			_ = getCmd01Context(start).Run()
			_ = getCmd02Context(start).Run()
			_ = getCmd03Context(start).Run()
		}
	}()

	connection = Connection{MayLog: true}
	connection.Connect(getDsn())
	reCreateIndex(t)
	insert(t)
	search(t)
}

func getDsn() string {
	if dsn := os.Getenv("ELASTICSEARCH_DSN"); dsn != "" {
		return dsn
	}

	return "http://127.0.0.50:9200,http://127.0.0.50:9202,http://127.0.0.50:9203"
}

func getPartialDsn() []string {
	if dsn := os.Getenv("ELASTICSEARCH_DSN"); dsn != "" {
		return strings.Split(dsn, ",")
	}

	return []string{"http://127.0.0.50:9200", "http://127.0.0.50:9202", "http://127.0.0.50:9203"}
}

func getCmd01Context(action string) *exec.Cmd {
	return exec.Command(sudo, user, docker, action, container01)
}

func getCmd02Context(action string) *exec.Cmd {
	return exec.Command(sudo, user, docker, action, container02)
}

func getCmd03Context(action string) *exec.Cmd {
	return exec.Command(sudo, user, docker, action, container03)
}

func reCreateIndex(t *testing.T) {
	_, err := connection.Client.Indices.Delete([]string{index})
	assert.Nil(t, err)

	body, _ := jsonEncode(map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"created": map[string]interface{}{
					"type":   "date",
					"format": "strict_date_optional_time||epoch_second",
				},
			},
		},
	})

	_, err = connection.Client.Indices.Create(index, connection.Client.Indices.Create.WithBody(&body))
	assert.Nil(t, err)
}

func insert(t *testing.T) {
	body, _ := jsonEncode(map[string]interface{}{
		"id":      "id",
		"created": "2010-10-10T10:10:10Z",
	})

	request := esapi.IndexRequest{
		Index:   index,
		Body:    strings.NewReader(body.String()),
		Refresh: "true",
	}

	_, err := request.Do(context.Background(), connection.Client)
	assert.Nil(t, err)
}

func search(t *testing.T) {
	body, _ := jsonEncode(map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	})

	response, err := connection.Client.Search(
		connection.Client.Search.WithIndex(index),
		connection.Client.Search.WithBody(&body),
	)

	assert.Nil(t, err)
	assert.False(t, response.IsError())
	processedResponse := processResponse(response)
	assert.Equal(t, map[string]interface{}{
		"id":      "id",
		"created": "2010-10-10T10:10:10Z",
	}, processedResponse["hits"].(map[string]interface{})["hits"].([]interface{})[0].(map[string]interface{})["_source"])
}

func jsonEncode(content interface{}) (bytes.Buffer, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(content); err != nil {
		return body, err
	}

	return body, nil
}

func processResponse(response *esapi.Response) map[string]interface{} {
	var body map[string]interface{}

	_ = json.NewDecoder(response.Body).Decode(&body)

	return body
}
