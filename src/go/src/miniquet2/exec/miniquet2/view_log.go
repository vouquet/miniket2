package main

import (
	"sync"
)

import (
	"github.com/nsf/termbox-go"
)

type LogViewLayer struct {
	ViewLayerBase
}

func NewLogViewLayer(strach_factor int) *LogViewLayer {
	return &LogViewLayer{
		ViewLayerBase{strach_factor:strach_factor, mtx:new(sync.Mutex)},
	}
}

func (self *LogViewLayer) SetValues(logs []*LogValue) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.call_flusher()

	r_size := self.tail - self.head
	if len(logs) > r_size {
		for i := 0; i <= r_size; i++ {
			log := logs[len(logs) - 1 - i]
			y := self.tail - i

			if y < self.head {
				break
			}
			self.setLine(log, y)
		}
		return
	}

	for i, log := range logs {
		y := self.head + i
		self.setLine(log, y)
	}
	sp_head := len(logs) + self.head
	for y := sp_head; y <= self.tail; y++ {
		self.setSpace(y)
	}
}

func (self *LogViewLayer) setLine(log *LogValue, y int) {
	var fg termbox.Attribute = termbox.ColorDefault

	if log.Type() == LogTypeErr {
		fg = termbox.ColorRed
	}

	for i, r := range log.Runes() {
		if i > self.width {
			return
		}

		self.call_setSell(i, y, r, fg, termbox.ColorDefault)
	}

	var space rune
	for i := len(log.Runes()); i < self.width; i++ {
		self.call_setSell(i, y, space, fg, termbox.ColorDefault)
	}
}
