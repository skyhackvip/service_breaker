package service_breaker

import (
	"errors"
	"log"
	"sync"
	"time"
)

var (
	ErrStateOpen    = errors.New("service breaker is open")
	ErrTooManyCalls = errors.New("service breaker is halfopen, too many calls")
)

//service breaker
type ServiceBreaker struct {
	mu               sync.RWMutex                                      //rwlock
	name             string                                            //name
	state            State                                             //current state
	windowInterval   time.Duration                                     //time window interval
	metrics          Metrics                                           //time window metrics statistics
	tripStrategyFunc TripStrategyFunc                                  //determine whether the breaker should be opened
	halfMaxCalls     uint64                                            //state is halfopen,max number of calls allowed.
	sleepTimeout     time.Duration                                     //state is open, after timeout will try to call.
	stateChangeHook  func(name string, fromState State, toState State) //hook if state change
	stateOpenTime    time.Time                                         //when state is open
}

//new breaker
func NewServiceBreaker(op Option) (*ServiceBreaker, error) {
	if op.WindowInterval <= 0 || op.HalfMaxCalls <= 0 || op.SleepTimeout <= 0 {
		return nil, errors.New("incomplete options")
	}
	breaker := new(ServiceBreaker)
	breaker.name = op.Name
	breaker.windowInterval = op.WindowInterval
	breaker.halfMaxCalls = op.HalfMaxCalls
	breaker.sleepTimeout = op.SleepTimeout
	breaker.stateChangeHook = op.StateChangeHook
	breaker.tripStrategyFunc = ChooseTrip(&op.TripStrategy)
	breaker.nextWindow(time.Now())
	return breaker, nil
}

//use breaker to call
func (breaker *ServiceBreaker) Call(exec func() (interface{}, error)) (interface{}, error) {
	log.Printf("start call, %v state is %v\n", breaker.name, breaker.state)
	//before call
	err := breaker.beforeCall()
	if err != nil {
		log.Printf("end call,%v batch:%v,metrics:(%v,%v,%v,%v,%v),window time start:%v\n\n",
			breaker.name,
			breaker.metrics.WindowBatch,
			breaker.metrics.CountAll,
			breaker.metrics.CountSuccess,
			breaker.metrics.CountFail,
			breaker.metrics.ConsecutiveSuccess,
			breaker.metrics.ConsecutiveFail,
			breaker.metrics.WindowTimeStart.Format("2006/01/02 15:04:05"))
		return nil, err
	}

	//if panic occur
	defer func() {
		err := recover()
		if err != nil {
			breaker.afterCall(false)
			panic(err) //todo?
		}
	}()

	//call
	breaker.metrics.OnCall()
	result, err := exec()

	//after call
	breaker.afterCall(err == nil)
	log.Printf("end call,%v batch:%v,metrics:(%v,%v,%v,%v,%v),window time start:%v\n\n",
		breaker.name,
		breaker.metrics.WindowBatch,
		breaker.metrics.CountAll,
		breaker.metrics.CountSuccess,
		breaker.metrics.CountFail,
		breaker.metrics.ConsecutiveSuccess,
		breaker.metrics.ConsecutiveFail,
		breaker.metrics.WindowTimeStart.Format("2006/1/2 15:04:05"))

	return result, err
}

//before intecept
func (breaker *ServiceBreaker) beforeCall() error {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()
	now := time.Now()
	switch breaker.state {
	case StateOpen:
		//after sleep timeout, can retry
		if breaker.stateOpenTime.Add(breaker.sleepTimeout).Before(now) {
			log.Printf("%s 熔断过冷却期，尝试半开\n", breaker.name)
			breaker.changeState(StateHalfOpen, now)
			return nil
		}
		log.Printf("%s 熔断打开，请求被阻止\n", breaker.name)
		return ErrStateOpen
	case StateHalfOpen:
		if breaker.metrics.CountAll >= breaker.halfMaxCalls {
			log.Printf("%s 熔断半开，请求过多被阻止\n", breaker.name)
			return ErrTooManyCalls
		}

	default: //Closed
		if !breaker.metrics.WindowTimeStart.IsZero() && breaker.metrics.WindowTimeStart.Before(now) {
			breaker.nextWindow(now)
			return nil
		}

	}
	return nil
}

//after intercept
func (breaker *ServiceBreaker) afterCall(success bool) {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	if success {
		breaker.onSuccess(time.Now())
	} else {
		breaker.onFail(time.Now())
	}

}

//call success
func (breaker *ServiceBreaker) onSuccess(now time.Time) {
	breaker.metrics.OnSuccess()
	if breaker.state == StateHalfOpen && breaker.metrics.ConsecutiveSuccess >= breaker.halfMaxCalls {
		breaker.changeState(StateClosed, now)
	}
}

//call fail
func (breaker *ServiceBreaker) onFail(now time.Time) {
	breaker.metrics.OnFail()
	switch breaker.state {
	case StateClosed:
		if breaker.tripStrategyFunc(breaker.metrics) {
			breaker.changeState(StateOpen, now)
		}
	case StateHalfOpen:
		breaker.changeState(StateOpen, now)

	}
}

//change breaker state
func (breaker *ServiceBreaker) changeState(state State, now time.Time) {
	if breaker.state == state {
		return
	}
	prevState := breaker.state
	breaker.state = state
	//goto next window,reset metrics
	breaker.nextWindow(time.Now())
	//record open time
	if state == StateOpen {
		breaker.stateOpenTime = now
	}
	//callback hook
	if breaker.stateChangeHook != nil {
		breaker.stateChangeHook(breaker.name, prevState, state)
	}
}

//goto next time window
func (breaker *ServiceBreaker) nextWindow(now time.Time) {
	breaker.metrics.NewBatch()
	breaker.metrics.OnReset() //clear count num
	var zero time.Time
	switch breaker.state {
	case StateClosed:
		if breaker.windowInterval == 0 {
			breaker.metrics.WindowTimeStart = zero
		} else {
			breaker.metrics.WindowTimeStart = now.Add(breaker.windowInterval)
		}
	case StateOpen:
		breaker.metrics.WindowTimeStart = now.Add(breaker.sleepTimeout)
	default: //halfopen
		breaker.metrics.WindowTimeStart = zero //halfopen no window
	}
}

func (breaker *ServiceBreaker) OpenBreaker() {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()
	if breaker.state == StateOpen {
		return
	}
	log.Printf("手工打开熔断器: %s\n", breaker.name)
	breaker.changeState(StateOpen, time.Now())
}

func (breaker *ServiceBreaker) CloseBreaker() {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()
	if breaker.state == StateClosed {
		return
	}
	log.Printf("手工关闭熔断器: %s\n", breaker.name)
	breaker.changeState(StateClosed, time.Now())
}

func (breaker *ServiceBreaker) State() State {
	breaker.mu.RLock()
	defer breaker.mu.RUnlock()
	return breaker.state
}
