package miniquet2

import (
	"time"
)

type Shop interface {
//	GetRate()   map[string]Rate
}

type Rate interface {
	Ask()       float64
	Bid()       float64
	High()      float64
	Last()      float64
	Low()       float64
	Symbol()    string
	Timestamp() time.Time
	Volume()    float64
}
