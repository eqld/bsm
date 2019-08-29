package supplier

import (
	"context"
	"io"
	"log"
	"net"
	"github.com/eqld/bsm/framebuffer"
)

// Supplier supplies given channel of framebuffer frames with data read from tcp connection.
type Supplier struct {
	listener net.Listener
	buffer   framebuffer.Buffer
	frames   chan<- framebuffer.Frame
}

// New returns new instance of a supplier.
func New(
	listener net.Listener,
	buffer framebuffer.Buffer,
	frames chan<- framebuffer.Frame,
) *Supplier {
	return &Supplier{
		listener: listener,
		buffer:   buffer,
		frames:   frames,
	}
}

// Serve makes a supplier to listen for input data and send in to a channel.
func (s *Supplier) Serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("fail to establish connection to input stream supplier: %v\n", err)
				return
			}

			// serve only one input stream supplier at a time
			s.serveInputConn(ctx, conn)
		}
	}
}

func (s *Supplier) serveInputConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr()
	log.Printf("serving %s as input stream supplier\n", remoteAddr)
	defer log.Printf("disconnecting input stream supplier %s\n", remoteAddr)

	var (
		f      framebuffer.Frame
		ok     bool
		n      int
		err    error
		closed bool
	)
	for !closed {
		f, ok = s.buffer.Get(ctx)
		if !ok {
			return
		}

		n, err = conn.Read(f.Data())
		if err == io.EOF {
			closed = true
			err = nil
		}
		if err != nil {
			log.Printf("error while reading a stream from input stream supplier %s: %v\n", remoteAddr, err)
			f.Use(-1)
			return
		}
		*f.Threshold() = n

		select {
		case s.frames <- f:
		case <-ctx.Done():
			f.Use(-1)
			return
		}
	}
}
