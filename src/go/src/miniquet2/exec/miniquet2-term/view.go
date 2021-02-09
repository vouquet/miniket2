package main

import (
	"fmt"
	"sync"
)

import (
	"github.com/nsf/termbox-go"
)

const (
	ColorSplitLineFG = termbox.Attribute(uint16(0xFD))
	ColorSplitLineBG = termbox.Attribute(uint16(0x12))
)

type View struct {
	vls  []ViewLayer

	width  int
	tail   int

	mtx *sync.Mutex
}

type ViewLayer interface {
	StratchFactor() int
	SetPosition(int, int, int)
	SetFlusher(func())
	SetCellWriter(func(int, int, rune, termbox.Attribute, termbox.Attribute))
	Title() string
}

func NewView(mode termbox.OutputMode,
		fg termbox.Attribute, bg termbox.Attribute) (*View, error) {
	self := &View{vls:make([]ViewLayer, 0), mtx:new(sync.Mutex)}
	self.init(mode, fg, bg)
	self.flush()

	self.resize()
	return self, nil
}

func (self *View) init(mode termbox.OutputMode,
		fg termbox.Attribute, bg termbox.Attribute) error {
	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.SetOutputMode(mode)
	termbox.Clear(fg, bg)
	return nil
}

func (self *View) SetTitle(t string) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.flush()

	var space rune = []rune("*")[0]
	self.setLine("== " + t + " ==", space, 0, ColorSplitLineFG, ColorSplitLineBG)
}

func (self *View) SetOperandMsg(s string, msg ...interface{}) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.flush()

	var space rune
	lstr := fmt.Sprintf(s, msg...)
	self.setLine(lstr, space, self.tail, termbox.ColorBlack, termbox.ColorWhite)
}

func (self *View) SetOperandErr(s string, msg ...interface{}) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.flush()

	var space rune
	lstr := fmt.Sprintf(s, msg...)
	self.setLine(lstr, space, self.tail, termbox.ColorWhite, termbox.ColorRed)
}

func (self *View) SetOperand(s string, msg ...interface{}) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	defer self.flush()

	var space rune
	var fg termbox.Attribute = termbox.ColorDefault
	lstr := fmt.Sprintf(s, msg...)
	self.setLine(lstr, space, self.tail, fg, termbox.ColorDefault)
}

func (self *View) Resize() {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.resize()
}

func (self *View) resize() {
	self.flush()
	x, y := termbox.Size()
	self.width = x
	self.tail = y - 1

	vls_range := y - 2

	den := 0
	for _, vl := range self.vls {
		den += vl.StratchFactor()
	}
	if den < 1 {
		return
	}

	size := vls_range / den
	rem := vls_range % den

	next_head := 1
	for i, vl := range self.vls {
		pos_range := size * vl.StratchFactor()
		if i == 0 {
			pos_range += rem
		}
		head := next_head
		tail_line := head + pos_range - 1
		tail := tail_line - 1

		vl.SetPosition(head, tail, x)
		self.setLayerTitle(vl.Title(), tail_line, x)

		next_head += pos_range
	}

	self.flush()
}

func (self *View) setLayerTitle(t string , y int, x int) {
	var space rune = []rune("-")[0]

	if t == "" {
		self.setLine("", space, y, ColorSplitLineFG, ColorSplitLineBG)
		return
	}
	self.setLine("--[" + t + "]--", space, y, ColorSplitLineFG, ColorSplitLineBG)
}

func (self *View) setLine(str string, space rune, y int,
								fg termbox.Attribute, bg termbox.Attribute) {
	wrote := 0

	if str != "" {
		for i, r := range []rune(str) {
			if i > self.width {
				return
			}

			termbox.SetCell(i, y, r, fg, bg)
			wrote++
		}
	}
	for i := wrote; i < self.width; i++ {
		termbox.SetCell(i, y, space, fg, bg)
	}
}

func (self *View) Flush() {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.flush()
}

func (self *View) flush() {
	termbox.Flush()
}

func (self *View) GetFuncPollEvent() func() termbox.Event {
	return termbox.PollEvent
}

func (self *View) AddViewLayer(vl ViewLayer) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	vl.SetFlusher(self.Flush)
	vl.SetCellWriter(termbox.SetCell)
	self.vls = append(self.vls, vl)
	self.resize()
}

func (self *View) Close() {
	termbox.Close()
}

type ViewLayerBase struct {
	strach_factor int
	title         string

	head  int
	tail  int
	width int

	flusher func()
	setSell func(int, int, rune, termbox.Attribute, termbox.Attribute)

	mtx   *sync.Mutex
}

func (self *ViewLayerBase) SetTitle(title string) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.title = title
}

func (self *ViewLayerBase) Title() string {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.title
}

func (self *ViewLayerBase) StratchFactor() int {
	return self.strach_factor
}

func (self *ViewLayerBase) SetPosition(head int, tail int, width int) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.head = head
	self.tail = tail
	self.width = width

	self.reset()
}

func (self *ViewLayerBase) SetCellWriter(f func(int, int, rune,
									termbox.Attribute, termbox.Attribute)) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.setSell = f
}

func (self *ViewLayerBase) call_setSell(x int, y int, r rune,
								fg termbox.Attribute, bg termbox.Attribute) {
	if self.setSell == nil {
		return
	}
	self.setSell(x, y, r, fg, bg)
}

func (self *ViewLayerBase) SetFlusher(f func()) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.flusher = f
}

func (self *ViewLayerBase) call_flusher() {
	if self.flusher == nil {
		return
	}
	self.flusher()
}

func (self *ViewLayerBase) reset() {
	for y := self.head; y <= self.tail; y++ {
		self.setSpace(y)
	}
}

func (self *ViewLayerBase) setSpace(y int) {
	var space rune

	for i := 0; i < self.width; i++ {
		self.call_setSell(i, y, space, termbox.ColorDefault, termbox.ColorDefault)
	}
}
