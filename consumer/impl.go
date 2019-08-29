package consumer

import (
	"context"
	"log"
	"net"
	"github.com/eqld/bsm/framebuffer"
)

// Consumer consumes framebuffer frames from given channel and sends data to tcp connections.
type Consumer struct {
	listener net.Listener
	frames   <-chan framebuffer.Frame
}

// New returns new instance of a consumer.
func New(
	listener net.Listener,
	frames <-chan framebuffer.Frame,
) *Consumer {
	return &Consumer{
		listener: listener,
		frames:   frames,
	}
}

type peer struct {
	seqNum   int
	conn     net.Conn
	frames   chan framebuffer.Frame
	closings chan<- int
}

func (p *peer) relay() {
	var err error
	for frame := range p.frames {
		_, err = p.conn.Write(frame.Data()[:*frame.Threshold()])
		frame.Use(-1)
		if err != nil {
			log.Printf("fail to send data frame to output stream consumer %d, disconnecting it: %v\n", p.seqNum, err)
			p.closings <- p.seqNum
			break
		}
	}
	// drain channel
	for frame := range p.frames {
		frame.Use(-1)
	}
}

// Serve makes a consumer to accept connections of output stream consumers and send data from given channel to them.
func (c *Consumer) Serve(ctx context.Context) {
	peers := make(chan *peer)
	closings := make(chan int)

	go c.servePeerPool(ctx, peers, closings)
	go c.acceptPeerConns(ctx, peers, closings)

	<-ctx.Done()
}

func (c *Consumer) servePeerPool(ctx context.Context, peers <-chan *peer, closings <-chan int) {
	const gcPeriod = 1024
	var gcCounter = 0

	pool := make(map[int]*peer)

	var (
		frame  framebuffer.Frame
		p      *peer
		seqNum int
		ok     bool
	)
	for {
		select {
		case p = <-peers:
			pool[p.seqNum] = p
		case seqNum = <-closings:
			if p, ok = pool[seqNum]; ok {
				close(p.frames)
				delete(pool, seqNum)
				gcCounter++
			}
		case frame = <-c.frames:
			frame.Use(len(pool))
			for _, p = range pool {
				p.frames <- frame
			}
			frame.Use(-1)
		case <-ctx.Done():
			return
		}

		if gcCounter >= gcPeriod {
			newPool := make(map[int]*peer)
			for k, v := range pool {
				newPool[k] = v
			}
			pool = newPool
			gcCounter = 0
		}
	}
}

func (c *Consumer) acceptPeerConns(ctx context.Context, peers chan<- *peer, closings chan<- int) {
	var i int
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := c.listener.Accept()
			if err != nil {
				log.Printf("fail to establish connection to output stream consumer: %v\n", err)
				return
			}

			p := &peer{
				seqNum:   i,
				conn:     conn,
				frames:   make(chan framebuffer.Frame, cap(c.frames)),
				closings: closings,
			}

			go p.relay()

			select {
			case peers <- p:
				log.Printf("serving %s as output stream consumer %d\n", p.conn.RemoteAddr(), p.seqNum)
			case <-ctx.Done():
				return
			}

			i++
		}
	}
}
