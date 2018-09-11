package light_poll

import (
	"errors"
)

var ErrClosed  = errors.New("poll is closed")

type Pool interface {
	Get() (interface{}, error)
	Put() (interface{}, error)
	Close() (interface{}, error)
	Release() ()
	Len() int
}

