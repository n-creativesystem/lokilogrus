package lokilogrus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Hook interface {
	logrus.Hook
	Stop()
	SetApp(name string)
}

type labelSets map[string]string

func (ls labelSets) Clone() labelSets {
	lsn := make(labelSets, len(ls))
	for ln, lv := range ls {
		lsn[ln] = lv
	}
	return lsn
}

type entries struct {
	Time string `json:"ts"`
	Line string `json:"line"`
}

type stream struct {
	Stream   map[string]string `json:"stream,omitempty"`
	Labels   string            `json:"labels,omitempty"`
	Value    [][2]string       `json:"values,omitempty"`
	Entities []entries         `json:"entries,omitempty"`
}

type data struct {
	Streams []stream `json:"streams"`
}

type client struct {
	url    *url.URL
	name   string
	labels labelSets
	log    *logrus.Logger

	stream chan stream
	wg     sync.WaitGroup
	quit   chan struct{}
	once   sync.Once
}

var _ Hook = (*client)(nil)

func New(log *logrus.Logger, app string, kv ...interface{}) (Hook, error) {
	logUrl := os.Getenv("LOG_URL")
	if logUrl == "" {
		logUrl = "http://localhost:3100"
	}
	u, err := url.Parse(logUrl)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "/loki/api/v1/push")
	labels := labelSets{}
	for i := 0; i < len(kv); i += 2 {
		labels[fmt.Sprintf("%v", kv[i])] = fmt.Sprintf("%v", kv[i+1])
	}
	baseLog := logrus.New()
	baseLog.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	if level, err := logrus.ParseLevel(os.Getenv("LOG_BASE_LEVEL")); err != nil {
		baseLog.SetLevel(logrus.InfoLevel)
	} else {
		baseLog.SetLevel(level)
	}
	client := &client{
		url:    u,
		name:   app,
		labels: labels,
		log:    baseLog,

		quit:   make(chan struct{}),
		stream: make(chan stream),
	}
	client.wg.Add(1)
	log.AddHook(client)
	go client.run()
	return client, nil
}

func (c *client) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	labels := c.labels.Clone()
	labels["app"] = c.name
	c.stream <- stream{
		Stream: labels,
		Value: [][2]string{
			{fmt.Sprintf("%d", time.Now().UnixNano()), line},
		},
	}
	return nil
}

func (c *client) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *client) send(s stream) {
	streams := data{
		Streams: []stream{s},
	}
	buf, _ := json.Marshal(&streams)
	req, _ := http.NewRequest(http.MethodPost, c.url.String(), bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.log.Error(err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 204 {
		_, _ = io.Copy(io.Discard, res.Body)
	} else {
		buf, _ = io.ReadAll(res.Body)
		c.log.WithField("status", res.StatusCode).Warnf("%s\n", string(buf))
	}
}

func (c *client) run() {
	defer c.wg.Done()
	for {
		select {
		case <-c.quit:
			return
		case s := <-c.stream:
			c.send(s)
		}
	}
}

func (c *client) Stop() {
	c.once.Do(func() { close(c.quit) })
	c.wg.Wait()
}

func (c *client) SetApp(name string) {
	c.name = name
}
