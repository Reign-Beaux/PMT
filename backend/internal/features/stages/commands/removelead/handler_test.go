package removelead

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	stage   domain.Stage
	getErr  error
	reads   int
	cleared bool
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	r.reads++
	if r.getErr != nil {
		return domain.Stage{}, r.getErr
	}
	s := r.stage
	s.ID = id
	return s, nil
}

func (r *fakeRepo) ClearLead(context.Context, uuid.UUID) error {
	r.cleared = true
	return nil
}

func TestHandle(t *testing.T) {
	pp := kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	withLead := func(status domain.Status) domain.Stage {
		return domain.Stage{Status: status, Lead: "tl", Members: []string{"dev"}}
	}
	tests := []struct {
		name   string
		actor  kernel.Actor
		stage  domain.Stage
		getErr error
		want   error
	}{
		{"PP quita en planning", pp, withLead(domain.StatusPlanning), nil, nil},
		{"usuario quita en stand_by", kernel.Owner, withLead(domain.StatusStandBy), nil, nil},
		{"etapa en in_dev", pp, withLead(domain.StatusInDev), nil, domain.ErrLeadNotRemovable},
		{"etapa en done", pp, withLead(domain.StatusDone), nil, domain.ErrLeadNotRemovable},
		{"etapa sin TL", pp, domain.Stage{Status: domain.StatusPlanning}, nil, domain.ErrStageWithoutLead},
		{"etapa inexistente o retirada", pp, domain.Stage{}, domain.ErrStageNotFound, domain.ErrStageNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{stage: tt.stage, getErr: tt.getErr}

			result, err := New(repo).Handle(context.Background(), Command{ID: uuid.New(), Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil {
				if !repo.cleared || result.Lead != "" || !slices.Equal(result.Members, []string{"dev"}) {
					t.Fatalf("cleared = %v, resultado = %+v", repo.cleared, result)
				}
				return
			}
			if repo.cleared {
				t.Fatal("no debió guardar")
			}
		})
	}
}

func TestHandleRejectsProfileBeforeReading(t *testing.T) {
	for _, profile := range []kernel.Profile{kernel.ProfileTL, kernel.ProfilePC, kernel.ProfileDP, kernel.ProfileDevBackend, kernel.ProfileQA} {
		t.Run(string(profile), func(t *testing.T) {
			repo := &fakeRepo{stage: domain.Stage{Status: domain.StatusPlanning, Lead: "tl"}}

			_, err := New(repo).Handle(context.Background(), Command{ID: uuid.New(), Actor: kernel.Actor{Name: "x", Profile: profile}})

			if !errors.Is(err, domain.ErrRemoveLeadForbidden) {
				t.Fatalf("err = %v", err)
			}
			if repo.reads != 0 || repo.cleared {
				t.Fatal("no debió leer ni guardar")
			}
		})
	}

}
