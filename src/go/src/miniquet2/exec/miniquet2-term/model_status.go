package main

import (
	"sync"
	"time"
	"strings"
)

import (
	"github.com/vouquet/shop"
)

type StatusModel struct {
	start_t    time.Time

	before     *StatusValue
	view_handler func(*StatusValue)

	mtx *sync.Mutex
}

func NewStatusModel() *StatusModel {
	return &StatusModel{start_t:time.Now(), mtx:new(sync.Mutex)}
}

func (self *StatusModel) ViewHandler(f func(*StatusValue)) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.view_handler = f
}

func (self *StatusModel) call_view_handler(sv *StatusValue) {
	if self.view_handler == nil {
		return
	}
	self.view_handler(sv)
}

func (self *StatusModel) Publish() {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if self.before == nil {
		return
	}
	self.call_view_handler(self.before)
}

func (self *StatusModel) UpdateStatus(rds map[string]shop.Rate) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	now := time.Now()
	uptime := time.Since(self.start_t)

	b_rs := make(map[string]*Rate)
	if self.before != nil {
		b_rs = self.before.Rates()
	}

	rates := make(map[string]*Rate)
	for _, rd := range rds {
		if strings.Contains(rd.Symbol(), "_JPY") {
			continue
		}

		r, ok := b_rs[rd.Symbol()]
		if !ok {
			rate, err := NewRate(rd, nil)
			if err != nil {
				continue
			}
			rates[rd.Symbol()] = rate
			continue
		}
		rate, err := NewRate(rd, r)
		if err != nil {
			continue
		}
		rates[rd.Symbol()] = rate
	}

	n_sv := &StatusValue{
		now: now,
		uptime: uptime,
		rates:rates,
	}
	self.before = n_sv
}

type StatusValue struct {
	now      time.Time
	uptime   time.Duration

	rates    map[string]*Rate
}

func (self *StatusValue) Now() time.Time {
	return self.now
}

func (self *StatusValue) Uptime() time.Duration {
	return self.uptime
}

func (self *StatusValue) Rates() map[string]*Rate {
	return self.rates
}

type Rate struct {
	symbol  string

	ask      float64
	ask_down bool
	ask_up   bool

	bid      float64
	bid_down bool
	bid_up   bool

	avg_day   float64
	avg_week  float64
	avg_month float64
}

func NewRate(r shop.Rate, before *Rate) (*Rate, error) {
	ask_down := false
	ask_up := false
	ask := r.Ask()
	if before != nil {
		if before.Ask() != ask {
			if before.Ask() < ask {
				ask_up = true
			} else {
				ask_down = true
			}
		}
	}

	bid_down := false
	bid_up := false
	bid := r.Bid()
	if before != nil {
		if before.Bid() != bid {
			if before.Bid() < bid {
				bid_up = true
			} else {
				bid_down = true
			}
		}
	}

	return &Rate{
		symbol: r.Symbol(),
		ask: ask,
		ask_up: ask_up,
		ask_down: ask_down,
		bid: bid,
		bid_up: bid_up,
		bid_down: bid_down,
	}, nil
}

func (self *Rate) Symbol() string {
	return self.symbol
}

func (self *Rate) Ask() float64 {
	return self.ask
}

func (self *Rate) AskUp() bool {
	return self.ask_up
}

func (self *Rate) AskDown() bool {
	return self.ask_down
}

func (self *Rate) Bid() float64 {
	return self.bid
}

func (self *Rate) BidUp() bool {
	return self.bid_up
}

func (self *Rate) BidDown() bool {
	return self.bid_down
}

func (self *Rate) AvgDay() float64 {
	return self.avg_day
}

func (self *Rate) AvgWeek() float64 {
	return self.avg_week
}

func (self *Rate) AvgMonth() float64 {
	return self.avg_month
}
