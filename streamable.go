package almon

// Streamable represents data that can be streamed to a channel
type Streamable interface {
	PrintForStream() []interface{}
	Print() string
}
