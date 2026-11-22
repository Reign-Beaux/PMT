package adoptorphans

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct{ users []uuid.UUID }

func (r *fakeRepo) AdoptOrphans(_ context.Context, userID uuid.UUID) (int64, error) {
	r.users = append(r.users, userID)
	return 2, nil
}

func TestHandle(t *testing.T) {
	repo := &fakeRepo{}
	owner := uuid.New()

	adopted, err := New(repo).Handle(context.Background(), owner)

	if err != nil || adopted != 2 || len(repo.users) != 1 || repo.users[0] != owner {
		t.Fatalf("adoptados = %d, usuarios = %v, err = %v", adopted, repo.users, err)
	}
}

func TestHandleWithoutOwner(t *testing.T) {
	repo := &fakeRepo{}

	if adopted, err := New(repo).Handle(context.Background(), uuid.Nil); err != nil || adopted != 0 || len(repo.users) != 0 {
		t.Fatalf("adoptados = %d, err = %v", adopted, err)
	}
}
