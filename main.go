package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flynn/flynn/discoverd/client"
	"github.com/flynn/flynn/pkg/stream"
	"gopkg.in/inconshreveable/log15.v2"
)

func main() {
	webhook := os.Getenv("SLACK_WEBHOOK")
	if webhook == "" {
		log.Fatal("missing SLACK_WEBHOOK")
	}

	services := strings.Split(os.Getenv("DISCOVERD_SERVICES"), ",")
	if len(services) == 0 {
		log.Fatal("missing DISCOVERD_SERVICES")
	}

	notifier := NewNotifier(webhook)
	for _, name := range services {
		if err := notifier.Watch(name); err != nil {
			log.Fatal(err)
		}
	}
	select {}
}

func NewNotifier(webhook string) *Notifier {
	n := &Notifier{webhook, make(chan *discoverd.Event, 1000)}
	go n.notifyLoop()
	return n
}

type Notifier struct {
	Webhook string
	Events  chan *discoverd.Event
}

func (n *Notifier) notifyLoop() {
	for event := range n.Events {
		if event.Instance == nil {
			continue
		}
		payload := struct {
			Username string `json:"username"`
			Text     string `json:"text"`
			Icon     string `json:"icon_emoji"`
		}{
			Username: "discoverd-notifier",
			Text: fmt.Sprintf(
				"%s: %s %s",
				strings.ToUpper(event.Kind.String()),
				event.Service,
				event.Instance.Meta["FLYNN_JOB_ID"],
			),
		}
		switch event.Kind {
		case discoverd.EventKindUp, discoverd.EventKindUpdate:
			payload.Icon = ":thumbsup:"
		case discoverd.EventKindDown:
			payload.Icon = ":thumbsdown:"
		}
		log15.Info("posting to Slack webhook", "text", payload.Text)
		data, _ := json.Marshal(payload)
		res, err := http.Post(n.Webhook, "application/json", bytes.NewReader(data))
		if err != nil {
			log15.Error("error posting to Slack webhook", "err", err)
			continue
		}
		res.Body.Close()
	}
}

func (n *Notifier) Watch(name string) error {
	log := log15.New("service", name)
	service := discoverd.NewService(name)
	var events chan *discoverd.Event
	var stream stream.Stream
	connect := func() (err error) {
		log.Info("connecting event stream")
		events = make(chan *discoverd.Event)
		stream, err = service.Watch(events)
		if err != nil {
			log.Error("error connecting event stream", "err", err)
		}
		return
	}
	if err := connect(); err != nil {
		return err
	}
	go func() {
		for {
			for {
				event, ok := <-events
				if !ok {
					break
				}
				select {
				case n.Events <- event:
				default:
					log.Warn("notifier channel overflow")
				}
			}
			log.Warn("event stream disconnected")
			for {
				if err := connect(); err == nil {
					break
				}
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}
