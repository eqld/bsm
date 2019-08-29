package framebuffer

import (
	"context"
)

// Buffer is interface for frame buffer.
type Buffer interface {
	// Get returns free frame from the buffer.
	// If there are no free frames, it blocks until at least one free frame appears.
	// Returned frame has one use. It is returned back to the buffer if amount of uses falls back to zero.
	// See `Frame.Use()` for details.
	Get(context.Context) (Frame, bool)

	// Stat returns current status of the buffer. See `Stat` for details.
	Stat() Stat

	// Close closes a frame buffer.
	Close() error
}

// Frame contains a slice of bytes from the buffer.
type Frame interface {

	// Data returns the underlying slice of bytes.
	Data() []byte

	// Threshold returns a pointer to a "threshold" value of a frame.
	// It can be used to set amount of valuable bytes of the frame.
	Threshold() *int

	// Use changes the amount of uses of the frame.
	// It is safe for concurrent use. If amount of uses falls to zero, frame is returned to the buffer.
	// After that the instance must be considered as irrevocably used and no method must be called further.
	// Usage of irrevocably used frame will definitely lead to undefined behaviour.
	// To get new free frame a client must call the method `Buffer.Get()`.
	// If amount of uses of a frame falls beyond zero, a panic in a separate goroutine is triggered,
	// so the client must care about proper handling of the uses, e.g. call `Frame.Use(-1)` in deferred function
	// just after `Buffer.Get()` and `Frame.Use(1)`.
	Use(int)
}

// Stat holds information about current status of the buffer: amount of total and used memory.
type Stat struct {
	Total int
	Used  int
}
