package framebuffer

import "context"

// New returns new instance of frame buffer implementation with given amount of frames of given size.
func New(size, count int) Buffer {
	mem := make([]byte, size*count)

	fs := &frames{
		frameSize: size,
		queue:     make(chan *frame, count),
		done:      make(chan struct{}),
	}

	for i := range make([]struct{}, count) {

		left := i * size
		right := left + size

		f := &frame{
			data:    mem[left:right],
			useChan: make(chan int),
			queue:   fs.queue,
		}

		go f.loop(fs.done)

		fs.queue <- f
	}

	return fs
}

type frames struct {
	frameSize int
	queue     chan *frame
	done      chan struct{}
}

func (fs *frames) Get(ctx context.Context) (f Frame, ok bool) {
	select {
	case f, ok = <-fs.queue:
	case <-ctx.Done():
	}

	if ok {
		f.Use(1)
	}

	return
}

func (fs *frames) Stat() Stat {
	free := len(fs.queue)
	total := cap(fs.queue)

	totalMem := total * fs.frameSize
	usedMem := (total - free) * fs.frameSize

	return Stat{
		Total: totalMem,
		Used:  usedMem,
	}
}

func (fs *frames) Close() error {
	close(fs.done)
	return nil
}

type frame struct {
	data      []byte
	threshold int
	uses      int
	useChan   chan int
	queue     chan<- *frame
}

func (f *frame) loop(done <-chan struct{}) {
	var n int
	for {
		select {
		case n = <-f.useChan:
			f.uses += n
		case <-done:
			return
		}

		if f.uses < 0 {
			// clients must use frame buffer properly to avoid setting that below zero
			panic("frame uses < 0")
		}

		if f.uses == 0 {
			f.queue <- f
		}
	}
}

func (f *frame) Use(n int) { f.useChan <- n }

func (f *frame) Data() []byte { return f.data }

func (f *frame) Threshold() *int { return &f.threshold }
