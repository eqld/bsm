package framebuffer

import (
	"context"
	"testing"
)

func Test_Buffer(t *testing.T) {
	ctx := context.Background()

	fs := New(3, 5)
	defer fs.Close()

	q := make(chan Frame, 5)

	// supplier
	go func() {
		for i := byte(0); i < 255; i++ {
			f, ok := fs.Get(ctx)
			if !ok {
				break
			}

			copy(f.Data(), []byte{i + 0, i + 1, i + 2})
			*f.Threshold() = int(f.Data()[2] % 3)

			q <- f
		}
		close(q)
	}()

	// consumers
	consume := func(t *testing.T, f Frame, i byte, dataExpected []byte, thresholdExpected int) {
		defer f.Use(-1)

		data := f.Data()
		threshold := *f.Threshold()

		if len(data) != len(dataExpected) {
			t.Errorf("%d: wrong length of frame data: got %d, want %d", i, len(data), len(dataExpected))
			return
		}

		for j := range dataExpected {
			if data[j] != dataExpected[j] {
				t.Errorf("%d: data corrupted: got '%v', want '%v'", i, data, dataExpected)
				return
			}
		}

		if threshold != thresholdExpected {
			t.Errorf("%d: wrong threshold value: got %d, want %d", i, threshold, thresholdExpected)
			return
		}
	}

	var i byte
	for f := range q {
		f := f

		dataExpected := []byte{i + 0, i + 1, i + 2}
		thresholdExpected := int(dataExpected[2] % 3)

		f.Use(3)
		go consume(t, f, i, dataExpected, thresholdExpected)
		go consume(t, f, i, dataExpected, thresholdExpected)
		go consume(t, f, i, dataExpected, thresholdExpected)

		f.Use(-1)
		i++
	}
}
