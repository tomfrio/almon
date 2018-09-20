package almon

// Publisher represents a publishing instance
type Publisher interface {
	GetName() string
	GetStream() chan Streamable
}

// StreamPublisher is an implementation of the Publisher protocol, using a channel stream
type StreamPublisher struct {
	Name   string
	Stream chan Streamable
}

// NewStreamPublisher returns a new StreamPublisher instance
func NewStreamPublisher(name string) *StreamPublisher {
	return &StreamPublisher{
		name,
		make(chan Streamable),
	}
}

// GetStream returns the publishing data stream
func (p *StreamPublisher) GetStream() chan Streamable {
	return p.Stream
}

// GetName returns the publishing name
func (p *StreamPublisher) GetName() string {
	return p.Name
}
