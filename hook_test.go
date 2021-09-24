package lokilogrus

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLokiHook(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.ExitFunc = func(code int) {
	}
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	h, err := New(log, "test", "key", "key-test")
	if !assert.NoError(t, err) {
		return
	}
	defer h.Stop()
	for i := 0; i < 10; i++ {
		cnt := i
		message := fmt.Sprintf("%s-%d", "これはテストです", cnt)
		log.Debug(message)
		time.Sleep(1 * time.Millisecond)
		log.Info(message)
		time.Sleep(1 * time.Millisecond)
		log.Warn(message)
		time.Sleep(1 * time.Millisecond)
		log.Error(message)
		time.Sleep(1 * time.Millisecond)
		log.Fatal(message)
		time.Sleep(1 * time.Millisecond)
	}
}

func TestStandard(t *testing.T) {
	std.ExitFunc = func(code int) {
	}
	for i := 0; i < 10; i++ {
		cnt := i
		message := fmt.Sprintf("%s-%d", "これはテストです", cnt)
		Debug(message)
		time.Sleep(1 * time.Millisecond)
		Info(message)
		time.Sleep(1 * time.Millisecond)
		Warn(message)
		time.Sleep(1 * time.Millisecond)
		Error(message)
		time.Sleep(1 * time.Millisecond)
		Fatal(message)
		time.Sleep(1 * time.Millisecond)
	}
}
