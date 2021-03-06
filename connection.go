package elasticsearch

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/hanaboso/go-log/pkg/zap"

	log "github.com/hanaboso/go-log/pkg"
)

type Connection struct {
	Client *elasticsearch.Client
	Log    log.Logger
	MayLog bool
}

func (connection *Connection) Connect(dsn string, timeout time.Duration, retries int) {
	var err error

	if connection.Log == nil {
		connection.Log = zap.NewLogger()
	}

	if connection.MayLog {
		connection.Client, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses:            strings.Split(dsn, ","),
			MaxRetries:           retries,
			EnableRetryOnTimeout: true,
			Transport: &http.Transport{
				ResponseHeaderTimeout: timeout,
				DialContext: (&net.Dialer{
					Timeout: timeout,
				}).DialContext,
			},
			Logger: &logger{Log: connection.Log},
		})
	} else {
		connection.Client, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses:            strings.Split(dsn, ","),
			MaxRetries:           retries,
			EnableRetryOnTimeout: true,
			Transport: &http.Transport{
				ResponseHeaderTimeout: timeout,
				DialContext: (&net.Dialer{
					Timeout: timeout,
				}).DialContext,
			},
		})
	}

	if err != nil {
		connection.logContext().Error(err)
		time.Sleep(time.Second)
		connection.Connect(dsn, timeout, retries)

		return
	}

	if _, err := connection.Client.Ping(); err != nil {
		connection.logContext().Error(err)
		time.Sleep(time.Second)
		connection.Connect(dsn, timeout, retries)

		return
	}
}

func (connection *Connection) Disconnect() {
	connection.Client = nil
}

func (connection *Connection) IsConnected() bool {
	if _, err := connection.Client.Ping(); err != nil {
		return false
	}

	return true
}

func (connection *Connection) logContext() log.Logger {
	return connection.Log.WithFields(map[string]interface{}{
		"package": "ElasticSearch",
	})
}

type logger struct {
	Log log.Logger
}

func (logger *logger) LogRoundTrip(request *http.Request, _ *http.Response, _ error, _ time.Time, duration time.Duration) error {
	if request != nil && request.Body != nil && request.Body != http.NoBody {
		if body, err := ioutil.ReadAll(request.Body); err == nil {
			logger.Log.WithFields(map[string]interface{}{
				"package": "ElasticSearch",
			}).Info("[%d ms] %s", duration.Milliseconds(), string(body))
		}
	}

	return nil
}

func (logger *logger) RequestBodyEnabled() bool {
	return true
}

func (logger *logger) ResponseBodyEnabled() bool {
	return false
}
