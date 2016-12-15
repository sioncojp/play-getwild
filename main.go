package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"golang.org/x/mobile/exp/audio/al"
	"golang.org/x/mobile/exp/f32"
)

var pctx *Context
var pianoPlayer *Piano

const (
	Pi         = float32(math.Pi)
	Fmt        = al.FormatStereo16
	QUEUE      = 500
	SampleRate = 10000 // 音の高さのベース
)

type Oscillator func() float32

type Context struct {
	sync.RWMutex
	source     al.Source
	queue      []al.Buffer
	oscillator Oscillator
}

type Piano struct {
	notes      []bool
	oscillator Oscillator
}

func G(gain float32, f Oscillator) Oscillator {
	return func() float32 {
		return gain * f()
	}
}

func GenOscillator(freq float32) Oscillator {
	dt := 1.0 / float32(SampleRate)
	k := 2.0 * Pi * freq
	T := 1.0 / freq
	t := float32(0.0)
	return func() float32 {
		res := f32.Sin(k * t)
		t += dt
		if t > T {
			t -= T
		}
		return res
	}
}

func Multiplex(fs ...Oscillator) Oscillator {
	return func() float32 {
		res := float32(0)
		for _, osc := range fs {
			res += osc()
		}
		return res
	}
}

func GenEnvelope(press *bool, f Oscillator) Oscillator {
	dt := 1.0 / float32(SampleRate)
	top := false
	gain := float32(0.0)
	attackd := dt / 0.01
	dekeyd := dt / 0.03
	sustainlevel := float32(0.3)
	sustaind := dt / 7.0
	released := dt / 0.8
	return func() float32 {
		if *press {
			if !top {
				gain += attackd
				if gain > 1.0 {
					top = true
					gain = 1.0
				}
			} else {
				if gain > sustainlevel {
					gain -= dekeyd
				} else {
					gain -= sustaind
				}
				if gain < 0.0 {
					gain = 0.0
				}
			}
		} else {
			top = false
			gain -= released
			if gain < 0.0 {
				gain = 0.0
			}
		}
		return gain * f()
	}
}

func NewContext(oscillator Oscillator) *Context {
	if err := al.OpenDevice(); err != nil {
		log.Fatal(err)
	}
	s := al.GenSources(1)
	return &Context{
		source:     s[0],
		queue:      []al.Buffer{},
		oscillator: oscillator,
	}
}

func NewPiano(freqs []float32) *Piano {
	p := new(Piano)
	p.notes = make([]bool, len(freqs))
	envelopes := []Oscillator{}
	for i, f := range freqs {
		base := []Oscillator{}
		for j := float32(1.0); j <= 8; j++ {
			base = append(base, G(0.5/j, GenOscillator(f*j)))
		}
		base = append(base, G(0.3, GenOscillator(f+2)))
		osc := Multiplex(base...)
		envelopes = append(envelopes, G(0.4, GenEnvelope(&p.notes[i], osc)))
	}
	p.oscillator = Multiplex(envelopes...) // all note oscilator multiplex
	return p
}
func (p *Piano) NoteOn(key int) {
	p.notes[key] = true
}

func (p *Piano) NoteOff(key int) {
	p.notes[key] = false
}

func (p *Piano) GetOscillator() Oscillator { return p.oscillator }

func (c *Context) Play(q int) {
	c.Lock()
	defer c.Unlock()
	n := c.source.BuffersProcessed()
	if n > 0 {
		rm := c.queue[:n]
		c.queue = nil
		c.source.UnqueueBuffers(rm...)
		al.DeleteBuffers(rm...)
	}
	fmt.Println(len(c.queue))
	for len(c.queue) < QUEUE {
		b := al.GenBuffers(q) // 音の長さ
		buf := make([]byte, 2048)
		for n := 0; n < 2048; n += 2 {
			f := c.oscillator()
			v := int16(float32(92767) * f) // 音の大きさ
			binary.LittleEndian.PutUint16(buf[n:n+2], uint16(v))
		}
		b[0].BufferData(Fmt, buf, SampleRate)
		c.source.QueueBuffers(b...)
		c.queue = append(c.queue, b...)
	}
	al.PlaySources(c.source)
}

func (c *Context) Close() {
	c.Lock()
	defer c.Unlock()
	al.StopSources(c.source)
}

func PlaySound(s, q int, slp time.Duration) {
	pianoPlayer.NoteOn(s)
	pctx.Play(q)
	time.Sleep(slp * time.Millisecond)
	pctx.Close()
	pianoPlayer.NoteOff(s)
}

func main() {
	pianoPlayer = NewPiano([]float32{
		246.941650628,
		261.625565301,
		277.182630977,
		293.664767917,
		311.126983722,
		329.627556913,
		349.228231433,
		369.994422712,
		391.995435982,
		415.30469758,
		440.0,
		466.163761518,
		493.883301256,
		523.251130601,
	})

	pctx = NewContext(pianoPlayer.GetOscillator())

	PlaySound(6, 70, 500)
	PlaySound(4, 70, 500)
	PlaySound(2, 10, 1000)

	time.Sleep(100 * time.Millisecond)

	PlaySound(6, 100, 300)
	PlaySound(4, 80, 500)
	PlaySound(2, 80, 500)
	PlaySound(2, 100, 300)
	PlaySound(2, 10, 1000)

	time.Sleep(200 * time.Millisecond)

	PlaySound(2, 100, 300)
	PlaySound(4, 150, 300)
	PlaySound(6, 150, 300)
	PlaySound(6, 150, 300)
	PlaySound(6, 150, 300)
	PlaySound(7, 150, 300)
	PlaySound(6, 150, 300)
	PlaySound(2, 150, 300)
	PlaySound(2, 150, 300)
	PlaySound(6, 150, 300)

	time.Sleep(10 * time.Millisecond)

	PlaySound(6, 80, 300)
	PlaySound(4, 50, 600)
	PlaySound(2, 200, 180)
	PlaySound(2, 10, 1000)
}
