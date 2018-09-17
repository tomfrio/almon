package streamer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
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

	fmt.Printf("`%s` connected.\n", client.Identifier)

	// notify the client
	event := Event{"info", 1, fmt.Sprintf("successfully connected as %s", client.Identifier)}
	client.events <- event

	return client
}

// Write broadcasted data to the client's stream
func (c *Client) write() {
	for {
		select {
		case streamable, ok := <-c.data:
			// stream data
			if !ok {
				c.closeConnection(errors.New("failed streaming data"))
				return
			}
			if err := c.socket.WriteMessage(websocket.TextMessage, []byte(streamable.Print())); err != nil {
				c.closeConnection(err)
				return
			}
		case event := <-c.events:
			// stream events
			jsonData, _ := json.Marshal(event)
			if err := c.socket.WriteJSON(string(jsonData)); err != nil {
				c.closeConnection(err)
				return
			}
		case <-time.After(1 * time.Second):
			// send heartbeat
			if err := c.socket.WriteMessage(websocket.TextMessage, []byte("<3")); err != nil {
				c.closeConnection(err)
				return
			}
		}
	}
}

// Listen for data being sent by the client
func (c *Client) listen() {
	for {
		_, msg, err := c.socket.ReadMessage()
		if err != nil {
			return
		}

		event, err := c.parseEvent(msg)
		if err != nil {
			continue
		}

		sub, err := c.parseSubscription(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}

		switch event.Event {
		case SUBSCRIBE:
			channel, err := c.hub.Subscribe(sub, c)
			if err != nil {
				event := Event{"error", 2, fmt.Sprintf("could not subscribe: %s", err)}
				c.events <- event
				continue
			}

			event := Event{"info", 1, fmt.Sprintf("successfully subscribed to %s", channel)}
			c.events <- event
		case UNSUBSCRIBE:
			channel, err := c.hub.Unsubscribe(sub, c)
			if err != nil {
				event := Event{"error", 2, fmt.Sprintf("could not unsubscribe: %s", err)}
				c.events <- event
				continue
			}

			event := Event{"info", 1, fmt.Sprintf("successfully unsubscribed from %s", channel)}
			c.events <- event
		}
	}
}

func (c *Client) closeConnection(err error) {
	c.socket.WriteMessage(websocket.CloseMessage, []byte{})
	c.socket.Close()

	fmt.Printf("`%s` disconnected: %s\n", c.Identifier, err)

	// unsubscribe client from all streams
	err = c.hub.UnsubscribeClient(c)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Client) parseSubscription(msg []byte) (SubscriptionEvent, error) {
	var sub SubscriptionEvent
	err := json.Unmarshal([]byte(msg), &sub)
	if err != nil {
		return sub, fmt.Errorf("could not parse subscription: %s", err)
	}

	return sub, nil
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
