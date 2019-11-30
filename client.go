package almon

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	readWait       = 60 * time.Second
	pingPeriod     = (readWait * 9) / 10
	maxMessageSize = 512
)

type clientSubscription struct {
	Name           string
	Specifications []string
}

// Client represents a client websocket connection
type Client struct {
	Identifier    string
	socket        *websocket.Conn
	data          chan Streamable
	events        chan Event
	hub           *Hub
	subscriptions []clientSubscription
	meta          map[string]interface{}
}

// NewClient returns a new Client instance
func NewClient(socket *websocket.Conn, hub *Hub, meta map[string]interface{}) *Client {
	identifier := randomString(6)

	var subs []clientSubscription
	client := &Client{
		identifier,
		socket,
		make(chan Streamable),
		make(chan Event),
		hub,
		subs,
		meta,
	}

	go client.write(hub.Writer)
	go client.listen()

	host, _ := os.Hostname()
	fmt.Printf("`%s` connected on `%s`.\n", client.Identifier, host)

	// notify the client
	client.events <- *NewEvent("info", 1, fmt.Sprintf("successfully connected as `%s`", client.Identifier))

	return client
}

// IsSubscribed checks whether a client is subscribed to a specific stream
func (c *Client) IsSubscribed(name string) bool {
	for _, sub := range c.subscriptions {
		if sub.Name == name {
			return true
		}
	}

	return false
}

// Subscribe a client to a channel
func (c *Client) Subscribe(metadata map[string]string) (string, error) {
	name, ok := metadata["channel"]
	if !ok {
		return name, errors.New("no channel specified")
	}

	_, err := c.hub.Attach(c)
	if err != nil {
		fmt.Println(err)
	}

	if c.IsSubscribed(name) {
		return name, fmt.Errorf("already subscribed to `%s`", name)
	}

	if !c.hub.HasStream(name) {
		return name, fmt.Errorf("channel `%s` does not exist", name)
	}

	c.subscriptions = append(c.subscriptions, clientSubscription{
		Name:           name,
		Specifications: []string{},
	})

	return name, nil
}

// Unsubscribe a client from a channel
func (c *Client) Unsubscribe(metadata map[string]string) (string, error) {
	name, ok := metadata["channel"]
	if !ok {
		return name, errors.New("no channel specified")
	}

	if !c.IsSubscribed(name) {
		return name, fmt.Errorf("not subscribed to `%s`", name)
	}

	// remove subscription
	subs, err := removeSub(c.subscriptions, name)
	if err != nil {
		fmt.Println(err)
	}
	c.subscriptions = subs

	if len(subs) < 1 {
		_, err := c.hub.Detach(c)
		if err != nil {
			fmt.Println(err)
		}
	}

	return name, nil
}

func removeSub(subs []clientSubscription, name string) ([]clientSubscription, error) {
	index := -1
	for i, sub := range subs {
		if sub.Name == name {
			index = i
			break
		}
	}

	if index < 0 {
		return subs, fmt.Errorf("no subscription for '%s' present", name)
	}

	// swap target with last
	subs[len(subs)-1], subs[index] = subs[index], subs[len(subs)-1]

	return subs[:len(subs)-1], nil
}

// Write broadcasted data to the client's stream
func (c *Client) write(wr Writer) {
	defer func() {
		c.closeConnection(errors.New("write error"), websocket.CloseAbnormalClosure)
	}()

	for {
		select {
		case streamable, ok := <-c.data:
			// stream data
			if !ok {
				return
			}
			if err := wr.WriteToClientStream(c, streamable); err != nil {
				return
			}
		case event := <-c.events:
			// stream events
			if err := wr.WriteToClientStream(c, event); err != nil {
				return
			}
		}
	}
}

// Listen for data being sent by the client
func (c *Client) listen() {
	defer func() {
		c.closeConnection(errors.New("read error"), websocket.CloseAbnormalClosure)
	}()

	for {
		_, msg, err := c.socket.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				// socket closed by client
				return
			}

			// cannot read from client
			return
		}

		received, err := ParseEventFromJSON(msg)
		if err != nil {
			// not parseable: skip
			continue
		}

		switch received.Event {
		case SUBSCRIBE:
			channel, err := c.Subscribe(received.Metadata)
			if err != nil {
				c.events <- *NewEvent("error", 2, fmt.Sprintf("could not subscribe: %s", err))
				continue
			}

			c.events <- *NewEvent("info", 1, fmt.Sprintf("successfully subscribed to `%s`", channel))
		case UNSUBSCRIBE:
			channel, err := c.Unsubscribe(received.Metadata)
			if err != nil {
				c.events <- *NewEvent("error", 2, fmt.Sprintf("could not unsubscribe: %s", err))
				continue
			}

			c.events <- *NewEvent("info", 1, fmt.Sprintf("successfully unsubscribed from `%s`", channel))
		default:
			c.hub.onEvent(received, c.meta)
		}
	}
}

func (c *Client) closeConnection(connErr error, closeCode int) {
	// detach client from the hub
	_, err := c.hub.Detach(c)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("`%s` disconnected.\n", c.Identifier)

	if closeCode != websocket.CloseAbnormalClosure {
		// close gracefully
		err = c.socket.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(closeCode, connErr.Error()),
			time.Now().Add(5*time.Second))
		if err != nil {
			fmt.Println(err)
		}
	}

	c.socket.Close()
}

func randomString(n int) string {
	letter := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)

	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}

	return string(b)
}
