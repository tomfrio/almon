package almon

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type eventHandler func(string, int, string, map[string]interface{})

// Hub contains streams for receiving and broadcasting data
type Hub struct {
	Publisher     Publisher
	Writer        Writer
	Subscribers   map[string]*Client
	EventHandler  eventHandler
	requiresToken bool
	sync.RWMutex
}

// NewHub returns a new Hub instance
func NewHub(pub Publisher, wr Writer) *Hub {
	return &Hub{
		Publisher:   pub,
		Writer:      wr,
		Subscribers: map[string]*Client{},
		EventHandler: func(event string, code int, message string, meta map[string]interface{}) {
			// fmt.Printf("unhandled event:\n%s\n%d\n%s\n%v\n", event, code, message, meta)
		},
		requiresToken: false,
	}
}

// Broadcast starts broadcasting
func (hub *Hub) Broadcast(port int) {
	// stream data
	for name := range hub.Publisher.GetStreams() {
		hub.Stream(name)
	}

	hub.listenForShutdown()

	// serve websockets
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleSocketConnection(w, r, hub)
	})

	fmt.Printf("listening on port %d\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// HasStream checks whether the hub has a specific streaming channel
func (hub *Hub) HasStream(name string) bool {
	if _, err := hub.Publisher.GetStream(name); err != nil {
		return false
	}

	return true
}

// Stream handles the pub/sub streaming for a single stream
func (hub *Hub) Stream(name string) {
	output := make(chan Streamable)

	// stream published data
	go func(hub *Hub) {
		defer close(output)
		stream, err := hub.Publisher.GetStream(name)
		if err != nil {
			fmt.Printf("streaming error: %s\n", err)
			return
		}

		fmt.Printf("streaming %s\n", name)
		for published := range stream {
			output <- published
		}
	}(hub)

	// fan out to all subscribers
	go func(hub *Hub) {
		for out := range output {
			hub.RLock()
			for _, sub := range hub.GetSubscribers(name) {
				sub.data <- out
			}
			hub.RUnlock()
		}
	}(hub)
}

func (hub *Hub) listenForShutdown() {
	// listen for system signals
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func(hub *Hub) {
		select {
		case sig := <-ch:
			fmt.Printf("got `%s` signal. closing all connections.\n", sig)
			// TODO: close 'connections' rather than subscribers
			for _, sub := range hub.Subscribers {
				sub.closeConnection(errors.New("server going down"), websocket.CloseServiceRestart)
			}
			os.Exit(1)
		}
	}(hub)
}

// GetSubscribers gets all eligible subscribers for a given stream
func (hub *Hub) GetSubscribers(stream string) map[string]*Client {
	streamSubs := make(map[string]*Client)
	for name, sub := range hub.Subscribers {
		if sub.IsSubscribed(stream) {
			streamSubs[name] = sub
		}
	}

	return streamSubs
}

// Attach a client to the hub
func (hub *Hub) Attach(c *Client) (string, error) {
	hub.Lock()
	defer hub.Unlock()

	name := hub.Publisher.GetName()

	_, ok := hub.Subscribers[c.Identifier]
	if ok {
		return name, fmt.Errorf("already attached to `%s`", name)
	}

	hub.Subscribers[c.Identifier] = c
	return name, nil
}

// Detach a client from the hub
func (hub *Hub) Detach(c *Client) (string, error) {
	hub.Lock()
	defer hub.Unlock()

	name := hub.Publisher.GetName()

	_, ok := hub.Subscribers[c.Identifier]
	if !ok {
		return name, fmt.Errorf("not attached to `%s`", name)
	}

	delete(hub.Subscribers, c.Identifier)
	return name, nil
}

// SetEventHandler sets a new event handler for the hub
func (hub *Hub) SetEventHandler(h eventHandler) {
	hub.EventHandler = h
}

// RequireToken sets authentication tokens as required for the hub
func (hub *Hub) RequireToken() {
	hub.requiresToken = true
}

func (hub *Hub) onEvent(event Event, clientMeta map[string]interface{}) {
	meta := make(map[string]interface{})
	for k, v := range event.Metadata {
		meta[k] = v
	}
	for k, v := range clientMeta {
		meta[k] = v
	}

	hub.EventHandler(event.Event, event.Code, event.Message, meta)
}

func handleSocketConnection(res http.ResponseWriter, req *http.Request, hub *Hub) {
	meta := make(map[string]interface{})
	if hub.requiresToken {
		q := req.URL.Query()
		token, ok := q["token"]
		if !ok || len(token) < 1 {
			serverError(res, http.StatusUnauthorized)
			return
		}
		meta["token"] = token[0]
	}

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		serverError(res, http.StatusInternalServerError)
		return
	}

	// create a listening client
	NewClient(conn, hub, meta)
}

func serverError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
