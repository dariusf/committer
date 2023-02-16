package server

import (
	"fmt"
	"sync"
	"time"
)

var lock sync.Mutex

var normalStart time.Time
var normalTotal int64

var monitorStart time.Time
var monitorTotal int64

func NormalStart() {
	lock.Lock()
	normalStart = time.Now()
	lock.Unlock()
}

func NormalEnd() {
	lock.Lock()
	normalTotal += time.Since(normalStart).Microseconds()
	lock.Unlock()
}

func MonitorStart() {
	lock.Lock()
	monitorStart = time.Now()
	lock.Unlock()
}

func MonitorEnd() {
	lock.Lock()
	monitorTotal += time.Since(monitorStart).Microseconds()
	lock.Unlock()
}

func PrintOverhead() {
	lock.Lock()
	var f float64 = float64(monitorTotal) / float64(normalTotal)
	fmt.Printf("%d / %d = %.2f\n", monitorTotal, normalTotal, f)
	lock.Unlock()
}
