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

// Client represents a client websocket connection
type Client struct {
	Identifier string
	socket     *websocket.Conn
	data       chan Streamable
	events     chan Event
	hub        *Hub
}

// NewClient returns a new Client instance
func NewClient(socket *websocket.Conn, hub *Hub) *Client {
	identifier := randomString(6)

	client := &Client{
		identifier,
		socket,
		make(chan Streamable),
		make(chan Event),
		hub,
	}

	go client.write(hub.Writer)
	go client.listen()

	host, _ := os.Hostname()
	fmt.Printf("`%s` connected on `%s`.\n", client.Identifier, host)

	// notify the client
	client.events <- *NewEvent("info", 1, fmt.Sprintf("successfully connected as `%s`", client.Identifier))

	return client
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
			channel, err := c.hub.Subscribe(c)
			if err != nil {
				c.events <- *NewEvent("error", 2, fmt.Sprintf("could not subscribe: %s", err))
				continue
			}

			c.events <- *NewEvent("info", 1, fmt.Sprintf("successfully subscribed to `%s`", channel))
		case UNSUBSCRIBE:
			channel, err := c.hub.Unsubscribe(c)
			if err != nil {
				c.events <- *NewEvent("error", 2, fmt.Sprintf("could not unsubscribe: %s", err))
				continue
			}

			c.events <- *NewEvent("info", 1, fmt.Sprintf("successfully unsubscribed from `%s`", channel))
		}
	}
}

func (c *Client) closeConnection(connErr error, closeCode int) {
	// unsubscribe client from the hub
	_, err := c.hub.Unsubscribe(c)
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
