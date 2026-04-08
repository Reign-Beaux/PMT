package issue

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type Issue struct {
	id       shared.ID
	phaseID  shared.ID
	title    Title
	spec     string
	status   Status
	priority Priority
	createdAt time.Time
	updatedAt time.Time
}

func New(phaseID shared.ID, title Title) (Issue, error) {
	if phaseID.IsZero() {
		return Issue{}, ErrInvalidPhaseID
	}
	if !title.isValid() {
		return Issue{}, ErrInvalidTitle
	}
	now := time.Now()
	return Issue{
		id:        shared.NewID(),
		phaseID:   phaseID,
		title:     title,
		status:    StatusOpen,
		priority:  PriorityMedium,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute rebuilds an Issue from persisted data.
// Bypasses constructor validation — callers must ensure data integrity.
func Reconstitute(id, phaseID shared.ID, title Title, spec string, status Status, priority Priority, createdAt, updatedAt time.Time) Issue {
	return Issue{
		id:        id,
		phaseID:   phaseID,
		title:     title,
		spec:      spec,
		status:    status,
		priority:  priority,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (i Issue) ID() shared.ID        { return i.id }
func (i Issue) PhaseID() shared.ID   { return i.phaseID }
func (i Issue) Title() Title         { return i.title }
func (i Issue) Spec() string         { return i.spec }
func (i Issue) Status() Status       { return i.status }
func (i Issue) Priority() Priority   { return i.priority }
func (i Issue) CreatedAt() time.Time { return i.createdAt }
func (i Issue) UpdatedAt() time.Time { return i.updatedAt }

func (i *Issue) UpdateTitle(title Title) error {
	if !title.isValid() {
		return ErrInvalidTitle
	}
	i.title = title
	i.updatedAt = time.Now()
	return nil
}

func (i *Issue) SetSpec(spec string) {
	i.spec = spec
	i.updatedAt = time.Now()
}

func (i *Issue) SetPriority(p Priority) {
	i.priority = p
	i.updatedAt = time.Now()
}

// Transition moves the issue to the given status.
// Valid transitions:
//
//	open        → in_progress, closed
//	in_progress → done, open
//	done        → closed
//	closed      → (terminal)
func (i *Issue) Transition(next Status) error {
	if !i.canTransitionTo(next) {
		return ErrInvalidTransition
	}
	i.status = next
	i.updatedAt = time.Now()
	return nil
}

func (i Issue) canTransitionTo(next Status) bool {
	allowed := map[Status][]Status{
		StatusOpen:       {StatusInProgress, StatusClosed},
		StatusInProgress: {StatusDone, StatusOpen},
		StatusDone:       {StatusClosed},
		StatusClosed:     {},
	}
	for _, s := range allowed[i.status] {
		if s == next {
			return true
		}
	}
	return false
}
