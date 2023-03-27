package multierror

import (
	"fmt"
	"strings"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

type MultiError interface {
	Combine() error
	Add(errs ...error)
	All() []error
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
	errMsgFmt := "Multiple Errors:\n\n" + strings.Repeat("%w\n", len(m.errs))
	errs := list.NewListMapTo[error, any]().MapTo(m.errs, func(err error) any {
		return err
	})
	return fmt.Errorf(errMsgFmt, errs...)
}

func (m *multiError) Add(errs ...error) {
	m.errs = append(m.errs, errs...)
}

func (m *multiError) All() []error {
	return m.errs
}
