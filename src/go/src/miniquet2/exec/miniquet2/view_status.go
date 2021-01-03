package main

import (
	"fmt"
	"sort"
	"sync"
)

import (
	"github.com/nsf/termbox-go"
)

const (
	SIZE_SV_SYMBOL = 6
	SIZE_SV_RATE = 10
)

type StatusViewLayer struct {
	ViewLayerBase
}

func NewStatusViewLayer(strach_factor int) *StatusViewLayer {
	return &StatusViewLayer{
		ViewLayerBase{strach_factor:strach_factor, mtx:new(sync.Mutex)},
	}
}

func (self *StatusViewLayer) SetValues(sv *StatusValue) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.call_flusher()

	if sv == nil {
		return
	}

	limit_size := self.tail - self.head

	n_s := sv.Now().Format("2006/01/02 15:04:05")
	h_s := fmt.Sprintf("%s, uptime %s", n_s, sv.Uptime())
	size := 0
	self.setLine(h_s, self.head, termbox.ColorDefault, termbox.ColorDefault)

//	h := fmt.Sprintf("Coin |   BID     |    ASK     |     day       week      month")
	h := fmt.Sprintf("Coin |   BID     |    ASK     |")
	size++
	self.setLine(h, self.head + size, termbox.ColorBlack, termbox.ColorWhite)

	rates := sv.Rates()
	k_idx := []string{}
	for k, _ := range rates {
		k_idx = append(k_idx, k)
	}
	sort.SliceStable(k_idx, func(i, j int) bool { return k_idx[i] < k_idx[j] })

	for _, k := range k_idx {
		size++
		if size > limit_size {
			return
		}

		r, ok := rates[k]
		if !ok {
			continue
		}

		y := self.head + size
		var np int = 0

		s_s := r.Symbol()
		np = self.setBlock(np, SIZE_SV_SYMBOL, y, s_s + ",", termbox.ColorDefault)

		s_b := fmt.Sprintf("%.3f", r.Bid())
		var fg_b termbox.Attribute = termbox.ColorDefault
		if r.BidUp() {
			fg_b = termbox.ColorGreen
		}
		if r.BidDown() {
			fg_b = termbox.ColorRed
		}
		np = self.setBlock(np, SIZE_SV_RATE, y, s_b, fg_b)

		split := " / "
		np = self.setBlock(np, len(split), y, split, termbox.ColorDefault)

		s_a := fmt.Sprintf("%.3f", r.Ask())
		var fg_a termbox.Attribute = termbox.ColorDefault
		if r.AskUp() {
			fg_a = termbox.ColorRed
		}
		if r.AskDown() {
			fg_a = termbox.ColorGreen
		}
		np = self.setBlock(np, SIZE_SV_RATE, y, s_a, fg_a)

/* TODO: set average into struct.
		s_ah := "  avg ("
		s_ad := fmt.Sprintf("%.3f", r.AvgDay())
		s_aw := fmt.Sprintf("%.3f", r.AvgWeek())
		s_am := fmt.Sprintf("%.3f", r.AvgMonth())
		s_at := ")"
		np = self.setBlock(np, len(s_ah), y, s_ah, termbox.ColorDefault)
		np = self.setBlock(np, SIZE_SV_RATE, y, s_ad, termbox.ColorDefault)
		np = self.setBlock(np, SIZE_SV_RATE, y, s_aw, termbox.ColorDefault)
		np = self.setBlock(np, SIZE_SV_RATE, y, s_am, termbox.ColorDefault)
		self.setBlock(np, len(s_at), y, s_at, termbox.ColorDefault)
*/
	}
}

func (self *StatusViewLayer) setLine(l string, y int, fg termbox.Attribute, bg termbox.Attribute) {
	rs := []rune(l)
	for i, r := range rs {
		if i > self.width {
			return
		}

		self.call_setSell(i, y, r, fg, bg)
	}

	var space rune
	for i := len(rs); i < self.width; i++ {
		self.call_setSell(i, y, space, fg, bg)
	}
}

func (self *StatusViewLayer) setBlock(sp int, max int, y int, v string, fg termbox.Attribute) int {
	size := sp + max
	rs := []rune(v)
	for i, r := range rs {
		if i > max {
			return size
		}

		self.call_setSell(sp + i, y, r, fg, termbox.ColorDefault)
	}

	var space rune
	for i := len(rs); i < max; i++ {
		self.call_setSell(sp + i, y, space, fg, termbox.ColorDefault)
	}
	return size
}
