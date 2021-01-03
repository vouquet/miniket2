package miniquet

import (
	"fmt"
	"sync"
	"time"
	"strconv"
)

import (
	"github.com/google/uuid"
	"github.com/vouquet/go-gmo-coin/gomocoin"
)

type Trader struct {//TODO: Upgrade2interface
	name        string
	description string

	win         float64

	st          *Storage
	shop        *gomocoin.GoMOcoin

	entries     map[string]*Entry
	check       func(*Entry, float64, float64) bool

	mtx         *sync.Mutex
}

func NewTrader(name string, desc string, shop *gomocoin.GoMOcoin, st *Storage) *Trader {
	return &Trader{
		name: name,
		description: desc,
		st: st,
		shop: shop,
		entries: make(map[string]*Entry),
		check: nil,
		mtx: new(sync.Mutex),
	}
}

func DecodeTrader(b []byte) (*Trader, error) {
	return nil, nil
}

func (self *Trader) Encode() ([]byte, error) {
	return nil, nil
}

func (self *Trader) Name() string {
	return self.name
}

func (self *Trader) Description() string {
	return self.description
}

func (self *Trader) Win() float64 {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.win
}

func (self *Trader) Add(symbol string, size float64, want_rate float64) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	entry := NewEntry(self.name, symbol, size, want_rate)
	_, ok := self.entries[entry.Id()]
	if ok {
		return fmt.Errorf("New entry id is already exist. '%s'", entry.Id())
	}

	self.entries[entry.Id()] = entry
	if err := self.st.Put(entry); err != nil {
		return err
	}
	return nil
}

func (self *Trader) RequestAppend(entry *Entry) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	_, ok := self.entries[entry.Id()]
	if ok {
		return fmt.Errorf("New entry id is already exist. '%s'", entry.Id())
	}

	self.entries[entry.Id()] = entry
	return nil

}

func (self *Trader) RequestStop(id string) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	entry, ok := self.entries[id]
	if !ok {
		return fmt.Errorf("%s is not found", id)
	}

	entry.Lastone()
	return nil
}

func (self *Trader) RequestKill9(id string) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	entry, ok := self.entries[id]
	if !ok {
		return fmt.Errorf("%s is not found", id)
	}

	if err := self.st.Delete(entry); err != nil {
		return err
	}
	delete(self.entries, entry.Id())
	return nil
}

func (self *Trader) Entries() map[string]*Entry {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.entries
}

func (self *Trader) GetEntriy(id string) (*Entry, bool)  {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	v, ok := self.entries[id]
	return v, ok
}

func (self *Trader) SetCheckFunc(f func(*Entry, float64, float64) bool) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.check = f
}

func (self *Trader) Do(log Logger, rates map[string]*gomocoin.RateData) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if self.check == nil {
		log.WriteErrLog("trader has not check function. target is nil pointer.")
		return
	}

	for _, entry := range self.entries {
		rate, ok := rates[entry.Symbol]
		if !ok {
			log.WriteErrLog("Not found symbol : '%s'", entry.Symbol)
			continue
		}

		ask, err := strconv.ParseFloat(rate.Ask, 64)
		if err != nil {
			log.WriteErrLog("Failed convert to float from rate(%s). : '%s'", entry.Symbol, err)
			continue
		}
		bid, err := strconv.ParseFloat(rate.Bid, 64)
		if err != nil {
			log.WriteErrLog("Failed convert to float from rate(%s). : '%s'", entry.Symbol, err)
			continue
		}

		if !self.check(entry, ask, bid) {
			continue
		}

		o_id, err := self.do(entry, ask, bid)
		if err != nil {
			log.WriteErrLog("Failed the trade: '%s'", err)
			continue
		}
		log.WriteMsgLog("Trade!!!!!! entry: %s, order_id: '%s'", entry.Id(), o_id)
	}

	return
}

func (self *Trader) do(entry *Entry, ask float64, bid float64) (string, error) {
	now := time.Now()

	o_id, err := self.shop.Order(entry.Position, entry.Symbol, entry.Size, nil)
	if err != nil {
		return "", err
	}

	if entry.IsLastone() {
		if err := self.st.Delete(entry); err != nil {
			return "", err
		}
		delete(self.entries, entry.Id())

		return o_id, nil
	}

	entry.Turn(now, ask, bid)
	return o_id, self.st.Put(entry)
}

type Entry struct {
	Uuid          uuid.UUID
	Trader      string

	Symbol      string
	Position    string

	Size        float64
	Win         float64

	Last_fix_rate float64
	Last_fix_date time.Time

	Gb01        float64
	Gb02        float64
	Gb03        []byte
	Gb04        []byte

	Last_run    bool
}

func NewEntry(trader string, symbol string, size float64, want_rate float64) *Entry {
	uuid := uuid.New()

	self := &Entry{
		Trader: trader,
		Uuid: uuid,
		Symbol: symbol,
		Size: size,

		Position: gomocoin.SIDE_BUY,
		Last_fix_date: time.Now(),
		Last_fix_rate: want_rate,
		Last_run: false,
	}
	self.resetBuf()
	return self
}

func (self *Entry) Turn(now time.Time, ask float64, bid float64) {
	if self.Position == gomocoin.SIDE_SELL {
		self.Win += float64(bid * self.Size) - float64(self.Last_fix_rate * self.Size)
		self.Last_fix_rate = bid

		self.Position = gomocoin.SIDE_BUY

	} else {
		self.Win += float64(self.Last_fix_rate * self.Size) - float64(ask * self.Size)
		self.Last_fix_rate = ask

		self.Position = gomocoin.SIDE_SELL
	}

	self.Last_fix_date = now
	self.resetBuf()
}

func (self *Entry) resetBuf() {
	self.Gb01 = float64(0)
	self.Gb02 = float64(0)
	self.Gb03 = []byte{}
	self.Gb04 = []byte{}
}


func (self *Entry) Point() float64 {
	return float64(self.Gb02 / self.Last_fix_rate)
}

func (self *Entry) Lastone() {
	self.Last_run = true
}

func (self *Entry) IsLastone() bool {
	return self.Last_run
}

func (self *Entry) Id() string {
	return self.Uuid.String()
}

func (self *Entry) LastRate() float64 {
	return self.Last_fix_rate
}

func (self *Entry) LastDate() time.Time {
	return self.Last_fix_date
}
