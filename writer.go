package almon

// Writer represents a writing instance
type Writer interface {
	WriteToClientStream(c *Client, st Streamable) error
}

// JSONWriter is an implementation of the Writer protocol
type JSONWriter struct {
}

// NewJSONWriter returns a new JSONWriter instance
func NewJSONWriter() *JSONWriter {
	return &JSONWriter{}
}

// WriteToClientStream writes data to the client stream as JSON
func (wr *JSONWriter) WriteToClientStream(c *Client, st Streamable) error {
	// stream events
	if err := c.socket.WriteJSON(st); err != nil {
		return err
	}

	return nil
}
