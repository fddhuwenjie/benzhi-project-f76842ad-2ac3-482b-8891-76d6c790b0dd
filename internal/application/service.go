package application

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type Service struct {
	store     *persistence.Store
	evaluator *chaincheck.Evaluator
	now       func() time.Time
	newID     func(string) string
}

func New(store *persistence.Store, evaluator *chaincheck.Evaluator) *Service {
	return &Service{store: store, evaluator: evaluator, now: time.Now, newID: randomID}
}

func randomID(prefix string) string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(value)
}

type Actor struct {
	Name string
	Role string
}

type RevisionCommand struct {
	Revision int64 `json:"revision"`
}

type operationFailure struct {
	operation string
	err       error
}

func (e *operationFailure) Error() string {
	return fmt.Sprintf("%s失败: %s", e.operation, e.err)
}

func (e *operationFailure) Unwrap() error { return e.err }

func failedOperation(operation string, err error) error {
	return &operationFailure{operation: operation, err: err}
}
