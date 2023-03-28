package multierror

import (
	"fmt"
	"strings"

	"errors"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

type MultiError struct {
	error
}

func (m MultiError) Unwrap() error {
	return m.error
}

type MultiErrorBuilder interface {
	Build() error
	Add(errs ...error)
	All() []error
	AllNonNil() []error
}

type multiErrorBuilder struct {
	errs []error
}

func NewMultiErrorBuilder(errs ...error) MultiErrorBuilder {
	return &multiErrorBuilder{
		errs: errs,
	}
}

func (m *multiErrorBuilder) Build() error {
	errs := m.AllNonNil()

	switch len(errs) {
	case 0:
		return nil
	default:
		errMsgFmt := "Multiple Errors:\n\n" + strings.Repeat("%w\n", len(errs))
		errsAsAny := list.NewListMapTo[error, any]().MapTo(errs, func(err error) any {
			return err
		})
		return MultiError{
			error: fmt.Errorf(errMsgFmt, errsAsAny...),
		}
	}
}

func (m *multiErrorBuilder) Add(errs ...error) {
	m.errs = append(m.errs, errs...)
}

func (m *multiErrorBuilder) All() []error {
	return m.errs
}

func (m *multiErrorBuilder) AllNonNil() []error {
	return list.NewListFilter[error]().Filter(m.errs, func(err error) bool {
		return !errors.Is(err, nil)
	})
}
