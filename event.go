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

// SubscriptionEvent represents an event for subscribing to a channel
type SubscriptionEvent struct {
	Event     string `json:"event"`
	ChannelID int    `json:"channel,string"`
	StreamID  int    `json:"stream,string"`
}

func (e *Event) getEvent() string {
	return e.Event
}

func (e *SubscriptionEvent) getEvent() string {
	return e.Event
}
