package hash

import (
	"crypto/md5"
	"encoding/json"

	"github.com/instana/instana-agent-operator/pkg/or_die"
)

type hasher struct {
	or_die.OrDie[[]byte]
}

func (h *hasher) HashJsonOrDie(obj interface{}) string {
	jsonStr := h.ResultOrDie(
		func() ([]byte, error) {
			return json.Marshal(obj)
		},
	)
	hash := md5.Sum(jsonStr)
	return string(hash[:])
}

type JsonHasher interface {
	HashJsonOrDie(obj interface{}) string
}

func NewJsonHasher() JsonHasher {
	return &hasher{
		OrDie: or_die.New[[]byte](),
	}
}
