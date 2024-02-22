package recovery

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/instana/instana-agent-operator/pkg/multierror"
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
		errBuilder := multierror.NewMultiErrorBuilder(*err)

		verb := func() string {
			switch p.(type) {
			case error:
				return "%w"
			default:
				return "%v"
			}
		}()

		errBuilder.AddSingle(fmt.Errorf("caught panic: "+verb+"\n%s", p, debug.Stack()))

		*err = caughtPanic{
			error: errBuilder.Build(),
		}
	}
}
