package listids

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeReader struct {
	stageID uuid.UUID
	ids     []uuid.UUID
}

func (r *fakeReader) ListIDs(_ context.Context, stageID uuid.UUID) ([]uuid.UUID, error) {
	r.stageID = stageID
	return r.ids, nil
}

func TestHandleListsStageTasks(t *testing.T) {
	stageID := uuid.New()
	reader := &fakeReader{ids: []uuid.UUID{uuid.New()}}

	ids, err := New(reader).Handle(context.Background(), stageID)

	if err != nil || len(ids) != 1 || reader.stageID != stageID {
		t.Errorf("ids = %v, etapa = %s, err = %v", ids, reader.stageID, err)
	}
}
