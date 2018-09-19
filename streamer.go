package almon

import "sync"

// StreamerProvider represents a configured data stream
type StreamerProvider interface {
	GetStreamer() *Streamer
	GetStreams() *HubStreamer
	Stream()
}

// Streamer is a concrete streaming instance
type Streamer struct {
	ID      int
	Name    string
	streams *HubStreamer
	sync.RWMutex
}

// NewStreamer returns a new Streamer instance
func NewStreamer(id int, name string) *Streamer {
	packets := make(map[int]*Stream)

	// add packet streams here..
	packets[id] = NewStream()

	return &Streamer{
		ID:   id,
		Name: name,
		streams: &HubStreamer{
			Streams: packets,
		},
	}
}

// GetStreamer returns the streamer itself
func (str *Streamer) GetStreamer() *Streamer {
	return str
}

// GetStreams returns the streamer's streaming channels
func (str *Streamer) GetStreams() *HubStreamer {
	return str.streams
}
