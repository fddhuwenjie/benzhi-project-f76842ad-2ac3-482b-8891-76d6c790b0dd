package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Ping 探测底层数据存储是否可用。健康检查通过该方法实际确认数据库可达，
// 而非无条件报告健康，避免监控在数据库不可用时误判实例状态。
func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}
