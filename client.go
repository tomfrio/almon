package almon

import (
	"encoding/json"
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

	go client.write()
	go client.listen()

	host, _ := os.Hostname()
	fmt.Printf("`%s` connected on `%s`.\n", client.Identifier, host)

	// notify the client
	event := Event{"info", 1, fmt.Sprintf("successfully connected as `%s`", client.Identifier)}
	client.events <- event

	return client
}

// Write broadcasted data to the client's stream
func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case streamable, ok := <-c.data:
			// stream data
			if !ok {
				c.closeConnection(errors.New("write error"), websocket.CloseAbnormalClosure)
				return
			}
			if err := c.socket.WriteMessage(websocket.TextMessage, []byte(streamable.Print())); err != nil {
				c.closeConnection(err, websocket.CloseNormalClosure)
				return
			}
		case event := <-c.events:
			// stream events
			jsonData, _ := json.Marshal(event)
			if err := c.socket.WriteJSON(string(jsonData)); err != nil {
				c.closeConnection(err, websocket.CloseNormalClosure)
				return
			}
		case <-ticker.C:
			// send ping
			c.socket.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
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

	c.socket.SetReadLimit(maxMessageSize)
	c.socket.SetReadDeadline(time.Now().Add(readWait))
	c.socket.SetPongHandler(func(string) error {
		// pong received: reset deadline
		c.socket.SetReadDeadline(time.Now().Add(readWait))
		return nil
	})

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

		event, err := c.parseEvent(msg)
		if err != nil {
			// not parseable: skip
			continue
		}

		switch event.Event {
		case SUBSCRIBE:
			channel, err := c.hub.Subscribe(c)
			if err != nil {
				event := Event{"error", 2, fmt.Sprintf("could not subscribe: %s", err)}
				c.events <- event
				continue
			}

			event := Event{"info", 1, fmt.Sprintf("successfully subscribed to `%s`", channel)}
			c.events <- event
		case UNSUBSCRIBE:
			channel, err := c.hub.Unsubscribe(c)
			if err != nil {
				event := Event{"error", 2, fmt.Sprintf("could not unsubscribe: %s", err)}
				c.events <- event
				continue
			}

			event := Event{"info", 1, fmt.Sprintf("successfully unsubscribed from `%s`", channel)}
			c.events <- event
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

	if closeCode == websocket.CloseAbnormalClosure {
		// don't close gracefully
		return
	}

	// send close message
	err = c.socket.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(closeCode, connErr.Error()),
		time.Now().Add(5*time.Second))
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Client) parseEvent(msg []byte) (Event, error) {
	var event Event
	err := json.Unmarshal([]byte(msg), &event)
	if err != nil {
		return event, fmt.Errorf("could not parse event: %s", err)
	}

	return event, nil
}

func randomString(n int) string {
	letter := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)

	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}

	return string(b)
}
