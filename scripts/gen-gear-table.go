package scripts

import (
	"math/rand"
	"time"
)

type GT struct{}

var (
	masterTable = []uint64{}
	gearTable   = make(map[uint64]bool)
)

func add(val uint64) {
	if gearTable[val] {
		return
	}

	masterTable = append(masterTable, val)
	gearTable[val] = true
}

func Run() []uint64 {
	for i := 0; i < 256; i++ {
		s := rand.NewSource(time.Now().UnixNano())
		r := rand.New(s)
		add(r.Uint64())
	}

	return masterTable
}
