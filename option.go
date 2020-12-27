package service_breaker

import (
	"time"
)

type TripStrategyOption struct {
	Strategy                 uint
	ConsecutiveFailThreshold uint64
	FailThreshold            uint64
	FailRate                 float64
	MinCall                  uint64
}

type Option struct {
	Name            string
	WindowInterval  time.Duration
	HalfMaxCalls    uint64
	SleepTimeout    time.Duration
	StateChangeHook func(name string, fromState State, toState State)
	TripStrategy    TripStrategyOption
}
