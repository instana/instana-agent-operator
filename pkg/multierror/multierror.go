package multierror

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

type MultiError interface {
	Combine() error
	Add(errs ...error)
	All() []error
	AllNonNil() []error
}

type multiError struct {
	errs []error
}

func NewMultiError(errs ...error) MultiError {
	return &multiError{
		errs: errs,
	}
}

func (m *multiError) Combine() error {
	errs := m.AllNonNil()

	errMsgFmt := "Multiple Errors:\n\n" + strings.Repeat("%w\n", len(errs))
	errsAsAny := list.NewListMapTo[error, any]().MapTo(errs, func(err error) any {
		return err
	})
	return fmt.Errorf(errMsgFmt, errsAsAny...)
}

func (m *multiError) Add(errs ...error) {
	m.errs = append(m.errs, errs...)
}

func (m *multiError) All() []error {
	return m.errs
}

func (m *multiError) AllNonNil() []error {
	return list.NewListFilter[error]().Filter(m.errs, func(err error) bool {
		return !errors.Is(err, nil)
	})
}
