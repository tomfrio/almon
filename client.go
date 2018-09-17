package streamer

import "github.com/gorilla/websocket"

// Client represents a client websocket connection
type Client struct {
	Identifier string
	socket     *websocket.Conn
	data       chan Streamable
	events     chan Event
	hub        *Hub
}
