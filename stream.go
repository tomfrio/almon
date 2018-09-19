package almon

import (
	"errors"
	"sync"
)

// Streamable represents data that can be streamed to a channel
type Streamable interface {
	PrintForStream() []interface{}
	Print() string
}

// Stream is a datastream for publishing/subscribing
type Stream struct {
	Publisher   chan Streamable
	Subscribers map[string]chan<- Streamable
	*sync.RWMutex
}

// NewStream returns a new Stream instance
func NewStream() *Stream {
	return &Stream{
		make(chan Streamable),
		make(map[string]chan<- Streamable),
		&sync.RWMutex{},
	}
}

// Broadcast the data on the stream to the subscribers
func (st *Stream) Broadcast() {
	output := make(chan Streamable)

	// stream published data
	go func() {
		defer close(output)
		for published := range st.Publisher {
			output <- published
		}
	}()

	// fan out to all subscribers
	go func() {
		for out := range output {
			st.RLock()
			for _, sub := range st.Subscribers {
				sub <- out
			}
			st.RUnlock()
		}
	}()
}

// Subscribe a client to the stream
func (st *Stream) Subscribe(id string, sub chan Streamable) error {
	st.Lock()
	defer st.Unlock()

	_, ok := st.Subscribers[id]
	if ok {
		return errors.New("already subscribed to channel")
	}
	st.Subscribers[id] = sub
	return nil
}

// Unsubscribe a client from the stream
func (st *Stream) Unsubscribe(id string) error {
	st.Lock()
	defer st.Unlock()

	_, ok := st.Subscribers[id]
	if !ok {
		return errors.New("not subscribed to channel")
	}
	delete(st.Subscribers, id)
	return nil
}

// Close all channels
func (st *Stream) Close() {
	close(st.Publisher)

	for _, pub := range st.Subscribers {
		close(pub)
	}
}
