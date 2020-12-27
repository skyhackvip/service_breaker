package service_breaker

import (
	"time"
)

//metrics
type Metrics struct {
	WindowBatch        uint64
	WindowTimeStart    time.Time
	CountAll           uint64
	CountSuccess       uint64
	CountFail          uint64
	ConsecutiveSuccess uint64
	ConsecutiveFail    uint64
}

func (m *Metrics) NewBatch() {
	m.WindowBatch++
}

func (m *Metrics) OnCall() {
	m.CountAll++
}

func (m *Metrics) OnSuccess() {
	m.CountSuccess++
	m.ConsecutiveSuccess++
	m.ConsecutiveFail = 0

}

func (m *Metrics) OnFail() {
	m.CountFail++
	m.ConsecutiveFail++
	m.ConsecutiveSuccess = 0
}

func (m *Metrics) OnReset() {
	m.CountAll = 0
	m.CountSuccess = 0
	m.CountFail = 0
	m.ConsecutiveSuccess = 0
	m.ConsecutiveFail = 0
}
