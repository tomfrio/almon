package almon

import (
	"encoding/json"
	"fmt"
)

// Possible events
const (
	SUBSCRIBE   = "subscribe"
	UNSUBSCRIBE = "unsubscribe"
)

// StreamableEvent represents an event that can be streamed to a channel
type StreamableEvent interface {
	getEvent() string
}

// Event represents either an `info`- or an `error` event
type Event struct {
	Event    string            `json:"event"`
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"meta"`
	ByteStreamed
}

// NewEvent creates a new event instance
func NewEvent(event string, code int, message string) *Event {
	return &Event{
		Event:   event,
		Code:    code,
		Message: message,
	}
}

// ParseEventFromJSON parses an event instance from json data
func ParseEventFromJSON(msg []byte) (Event, error) {
	var event Event
	err := json.Unmarshal(msg, &event)
	if err != nil {
		return event, fmt.Errorf("could not parse event: %s", err)
	}

	return event, nil
}

func (e *Event) getEvent() string {
	return e.Event
}
