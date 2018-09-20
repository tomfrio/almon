package almon

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
	Event   string `json:"event"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Event) getEvent() string {
	return e.Event
}
