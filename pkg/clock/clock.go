package clock

import "time"

type Clock struct{}

func New() *Clock {
	return &Clock{}
}

func (*Clock) Now() time.Time {
	return time.Now()
}
