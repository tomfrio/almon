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

// Hub contains streams for receiving and broadcasting data
type Hub struct {
	Publisher   Publisher
	Subscribers map[string]*Client
	sync.RWMutex
}

// NewHub returns a new Hub instance
func NewHub(pub Publisher) *Hub {
	return &Hub{
		Publisher:   pub,
		Subscribers: map[string]*Client{},
	}
}

// Broadcast starts broadcasting
func (hub *Hub) Broadcast(port int) {
	// stream data
	hub.Stream()

	// serve websockets
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleSocketConnection(w, r, hub)
	})

	fmt.Printf("listening on port %d\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// Stream handles the pub/sub streaming
func (hub *Hub) Stream() {
	output := make(chan Streamable)

	// stream published data
	go func(hub *Hub) {
		defer close(output)
		for published := range hub.Publisher.GetStream() {
			output <- published
		}
	}(hub)

	// fan out to all subscribers
	go func(hub *Hub) {
		for out := range output {
			hub.RLock()
			for _, sub := range hub.Subscribers {
				sub.data <- out
			}
			hub.RUnlock()
		}
	}(hub)

	// listen for system signals
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func(hub *Hub) {
		select {
		case sig := <-ch:
			fmt.Printf("got `%s` signal. closing all connections.\n", sig)
			// TODO: close 'connections' rather than subscribers
			for _, sub := range hub.Subscribers {
				sub.closeConnection(errors.New("server going down"), websocket.CloseAbnormalClosure)
			}
			os.Exit(1)
		}
	}(hub)
}

// Subscribe a client to the channel
func (hub *Hub) Subscribe(c *Client) (string, error) {
	hub.Lock()
	defer hub.Unlock()

	name := hub.Publisher.GetName()

	_, ok := hub.Subscribers[c.Identifier]
	if ok {
		return "", fmt.Errorf("already subscribed to `%s`", name)
	}

	hub.Subscribers[c.Identifier] = c
	return name, nil
}

// Unsubscribe a client from the channel
func (hub *Hub) Unsubscribe(c *Client) (string, error) {
	hub.Lock()
	defer hub.Unlock()

	name := hub.Publisher.GetName()

	_, ok := hub.Subscribers[c.Identifier]
	if !ok {
		return "", fmt.Errorf("not subscribed to `%s`", name)
	}

	delete(hub.Subscribers, c.Identifier)
	return name, nil
}

func handleSocketConnection(res http.ResponseWriter, req *http.Request, hub *Hub) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// create a listening client
	NewClient(conn, hub)
}
