package lofi

import (
	"math"
	"math/rand"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
)

type LowPassFilter struct {
	Streamer beep.Streamer
	Cutoff   float64
	Fs       float64
	prevL    float64
	prevR    float64
}

func (l *LowPassFilter) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = l.Streamer.Stream(samples)
	alpha := 2.0 * math.Pi * l.Cutoff / l.Fs
	if alpha > 1.0 {
		alpha = 1.0
	}
	for i := 0; i < n; i++ {
		l.prevL = l.prevL + alpha*(samples[i][0]-l.prevL)
		l.prevR = l.prevR + alpha*(samples[i][1]-l.prevR)
		samples[i][0] = l.prevL
		samples[i][1] = l.prevR
	}
	return n, ok
}

func (l *LowPassFilter) Err() error {
	return l.Streamer.Err()
}

type WhiteNoiseStreamer struct{}

func (w WhiteNoiseStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		val := rand.Float64()*2.0 - 1.0
		samples[i][0] = val
		samples[i][1] = val
	}
	return len(samples), true
}

func (w WhiteNoiseStreamer) Err() error {
	return nil
}

type DiskStreamer struct {
	file     *os.File
	streamer beep.StreamSeekCloser
}

func NewDiskStreamer(path string) (*DiskStreamer, beep.Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	s, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return nil, beep.Format{}, err
	}
	return &DiskStreamer{file: f, streamer: s}, format, nil
}

func (ds *DiskStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	return ds.streamer.Stream(samples)
}

func (ds *DiskStreamer) Err() error {
	return ds.streamer.Err()
}

func (ds *DiskStreamer) Close() error {
	ds.streamer.Close()
	return ds.file.Close()
}

func (ds *DiskStreamer) Len() int {
	return ds.streamer.Len()
}

func (ds *DiskStreamer) Position() int {
	return ds.streamer.Position()
}

func (ds *DiskStreamer) Seek(p int) error {
	return ds.streamer.Seek(p)
}
