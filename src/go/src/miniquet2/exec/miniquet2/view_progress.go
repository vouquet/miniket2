package main

import (
	"fmt"
	"sort"
	"sync"
)

import (
	"github.com/nsf/termbox-go"
)

type ProgressViewLayer struct {
	ViewLayerBase
}

func NewProgressViewLayer(strach_factor int) *ProgressViewLayer {
	return &ProgressViewLayer{
		ViewLayerBase{strach_factor:strach_factor, mtx:new(sync.Mutex)},
	}
}

func (self *ProgressViewLayer) SetValues(trs map[string]*Trader) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.call_flusher()

	limit_size := self.tail - self.head

	index := []string{}
	for k, _ := range trs {
		index = append(index, k)
	}
	sort.SliceStable(index, func(i, j int) bool { return index[i] < index[j] })

	size := 0
	for i, k := range index {
		if i != 0{
			size++
		}
		if size > limit_size {
			return
		}

		tr := trs[k]

		y := self.head + size
		np := 0
		np = self.setBlock(np, 2, y, "* ", termbox.ColorDefault)
		np = self.setBlock(np, 5, y, tr.Name(), termbox.ColorDefault)
		d_runes := []rune(tr.Description())
		if len(d_runes) > 97 {
			d_runes = append(d_runes[:96], []rune("...")...)
		}
		d_str := fmt.Sprintf(" (%s), ", string(d_runes))
		np = self.setBlock(np, 100, y, d_str, termbox.ColorDefault)
		//w_str := fmt.Sprintf("WIN : %.3f", tr.Win()) //TODO: add function
		//self.setBlock(np, 10, y, w_str, termbox.ColorDefault)

		en_index := []string{}
		for k, _ := range tr.Entries() {
			en_index = append(en_index, k)
		}
		sort.SliceStable(en_index, func(i, j int) bool { return en_index[i] < en_index[j] })

		for _, id := range en_index {
			size++
			if size > limit_size {
				return
			}

			y := self.head + size
			np := 0
			np = self.setBlock(np, 5, y, "  ┠- ", termbox.ColorDefault)

			en, ok := tr.GetEntriy(id)
			if !ok {
				self.setBlock(np, 37, y, k + "(data not found)", termbox.ColorDefault)
				continue
			}

			np = self.setBlock(np, 37, y, en.Id(), termbox.ColorDefault)

			size_str := fmt.Sprintf("%.5f", en.Size)
			np = self.setBlock(np, 6, y, " [" + en.Position, termbox.ColorDefault)
			np = self.setBlock(np, 16, y, ":" + en.Symbol + "(" + size_str + ")] ", termbox.ColorDefault)

			point_str := fmt.Sprintf("%.10f", en.Point())
			np = self.setBlock(np, 18, y, "Point: " + point_str, termbox.ColorDefault)

			win_str := fmt.Sprintf("%.3f ", en.Win)
			var win_color termbox.Attribute = termbox.ColorDefault
			if float64(0) < en.Win {
				win_color = termbox.ColorGreen
			}
			if float64(0) > en.Win {
				win_color = termbox.ColorRed
			}
			np = self.setBlock(np, 6, y, " Win: ", termbox.ColorDefault)
			np = self.setBlock(np, 9, y, win_str, win_color)

			lt_s := en.LastDate().Format("2006-01-02 15:04:05")
			n_str := fmt.Sprintf("LastOrder{Rate: %.3f, Date: %s}", en.LastRate(), lt_s)
			np = self.setBlock(np, 100, y, n_str, termbox.ColorDefault)
		}
	}

	return
}

func (self *ProgressViewLayer) setLine(l string, y int, fg termbox.Attribute, bg termbox.Attribute) {
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

func (self *ProgressViewLayer) setBlock(sp int, max int, y int, v string, fg termbox.Attribute) int {
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
