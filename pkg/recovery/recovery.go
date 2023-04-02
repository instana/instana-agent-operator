package recovery

import (
	"errors"
	"fmt"
	"runtime/debug"
)

type caughtPanic struct {
	error
}

func (c caughtPanic) Unwrap() error {
	return c.error
}

func AsCaughtPanic(err error) bool {
	return errors.As(err, &caughtPanic{})
}

func Catch(err *error) {
	if p := recover(); p != nil {
		verb := func() string {
			switch p.(type) {
			case error:
				return "%w"
			default:
				return "%v"
			}
		}()
		*err = func() error {
			return caughtPanic{
				error: fmt.Errorf("caught panic: "+verb+"\n%s", p, debug.Stack()),
			}
		}()
	}
}
