/*
 * (c) Copyright IBM Corp. 2024, 2026
 * (c) Copyright Instana Inc. 2024, 2026
 */

package hash

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"

	"github.com/instana/instana-agent-operator/pkg/or_die"
)

type hasher struct {
	or_die.OrDie[[]byte]
}

func (h *hasher) HashJsonOrDie(obj any) string {
	jsonStr := h.ResultOrDie(
		func() ([]byte, error) {
			return json.Marshal(obj)
		},
	)
	hash := md5.Sum(jsonStr)
	return base64.StdEncoding.EncodeToString(hash[:])
}

type JsonHasher interface {
	HashJsonOrDie(obj any) string
}

func NewJsonHasher() JsonHasher {
	return &hasher{
		OrDie: or_die.New[[]byte](),
	}
}
