package almon

import "fmt"

// Publisher represents a publishing instance
type Publisher interface {
	GetName() string
	GetStreams() map[string]chan Streamable
	GetStream(string) (chan Streamable, error)
}

// StreamPublisher is an implementation of the Publisher protocol, using a channel stream
type StreamPublisher struct {
	Name    string
	Streams map[string]chan Streamable
}

// NewStreamPublisher returns a new StreamPublisher instance
func NewStreamPublisher(name string, streams []string) *StreamPublisher {
	stm := make(map[string]chan Streamable, len(streams))
	for _, name := range streams {
		stm[name] = make(chan Streamable)
	}

	return &StreamPublisher{name, stm}
}

// GetStreams returns all available data streams
func (p *StreamPublisher) GetStreams() map[string]chan Streamable {
	return p.Streams
}

// GetStream returns a publishing data stream by it's name
func (p *StreamPublisher) GetStream(name string) (chan Streamable, error) {
	stream, ok := p.Streams[name]
	if !ok {
		return nil, fmt.Errorf("stream '%s' not available", name)
	}

	return stream, nil
}

// GetName returns the publisher name
func (p *StreamPublisher) GetName() string {
	return p.Name
}
