package multierror

import (
	"errors"
	"fmt"
	"strings"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

type multiError struct {
	error
}

func (m multiError) Unwrap() error {
	return m.error
}

func AsMultiError(err error) bool {
	return errors.As(err, &multiError{})
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
	case 1:
		return multiError{
			error: errs[0],
		}
	default:
		errMsgFmt := "Multiple Errors:\n" + strings.Repeat("%w\n", len(errs))
		errsAsAny := list.NewListMapTo[error, any]().MapTo(
			errs, func(err error) any {
				return err
			},
		)
		return multiError{
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
	return list.NewListFilter[error]().Filter(
		m.errs, func(err error) bool {
			return !errors.Is(err, nil)
		},
	)
}
