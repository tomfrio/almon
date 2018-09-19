package almon

import (
	"errors"
	"fmt"
	"sync"
)

// HubStreamer contains separated streams for a hub
type HubStreamer struct {
	Streams map[int]*Stream
}

// Hub contains streams for receiving and broadcasting data
type Hub struct {
	Streamers map[int]*HubStreamer
	sync.RWMutex
}

// NewHub returns a new Hub instance
func NewHub() *Hub {
	return &Hub{
		Streamers: make(map[int]*HubStreamer),
	}
}

// AddStreamer adds a new streamer to the hub
func (hub *Hub) AddStreamer(id int, streamer *HubStreamer) {
	hub.Streamers[id] = streamer
}

// Broadcast the data on all streamers to their subscribers
func (hub *Hub) Broadcast() {
	for _, streamer := range hub.Streamers {
		for _, stream := range streamer.Streams {
			stream.Broadcast()
		}
	}
}

// Subscribe a client to a stream
func (hub *Hub) Subscribe(event SubscriptionEvent, c *Client) (string, error) {
	stream, err := hub.getStreamForEvent(event)
	if err != nil {
		return "", err
	}

	err = stream.Subscribe(c.Identifier, c.data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("channel-%d-%d", event.ChannelID, event.StreamID), nil
}

// Unsubscribe a client from a stream
func (hub *Hub) Unsubscribe(event SubscriptionEvent, c *Client) (string, error) {
	stream, err := hub.getStreamForEvent(event)
	if err != nil {
		return "", err
	}

	err = stream.Unsubscribe(c.Identifier)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("channel-%d-%d", event.ChannelID, event.StreamID), nil
}

// UnsubscribeClient unsubscribes a client from all stream
func (hub *Hub) UnsubscribeClient(c *Client) error {
	i := 0
	for _, streamers := range hub.Streamers {
		for _, stream := range streamers.Streams {
			_, ok := stream.Subscribers[c.Identifier]
			if ok {
				err := stream.Unsubscribe(c.Identifier)
				if err != nil {
					return err
				}
				i++
			}
		}
	}

	fmt.Printf("`%s` unsubscribed from %d channel(s).\n", c.Identifier, i)
	return nil
}

func (hub *Hub) getStreamForEvent(event SubscriptionEvent) (*Stream, error) {
	if event.ChannelID < 1 {
		return nil, errors.New("no channel supplied")
	}

	if event.StreamID < 1 {
		return nil, errors.New("no stream supplied")
	}

	streamer, ok := hub.Streamers[event.ChannelID]
	if !ok {
		return nil, fmt.Errorf("no stream for channel %d available", event.ChannelID)
	}

	stream, ok := streamer.Streams[event.StreamID]
	if !ok {
		return nil, fmt.Errorf("stream %d not available", event.StreamID)
	}

	return stream, nil
}

// CloseChannels closes all subscribed and publishing channels on the streamer
func (hs *HubStreamer) CloseChannels() {
	for _, stream := range hs.Streams {
		stream.Close()
	}

	for _, stream := range hs.Streams {
		stream.Close()
	}
}
