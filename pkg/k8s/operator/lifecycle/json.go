package lifecycle

import (
	"encoding/json"

	"github.com/instana/instana-agent-operator/pkg/or_die"
)

type jsonMarshaler interface {
	marshalOrDie(obj any) []byte
	unMarshalOrDie(raw []byte, obj any)
}

type jsonOrDie struct {
	or_die.OrDie[[]byte]
}

func (j *jsonOrDie) marshalOrDie(obj any) []byte {
	return j.ResultOrDie(
		func() ([]byte, error) {
			return json.Marshal(obj)
		},
	)
}

func (j *jsonOrDie) unMarshalOrDie(raw []byte, obj any) {
	j.ResultOrDie(
		func() ([]byte, error) {
			return nil, json.Unmarshal(raw, obj)
		},
	)
}
