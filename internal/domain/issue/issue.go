package issue

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type Issue struct {
	id        shared.ID
	projectID shared.ID
	phaseID   *shared.ID // nil = backlog (not assigned to any phase)
	issueType IssueType
	title     Title
	spec      string
	status    Status
	priority  Priority
	dueDate   *time.Time
	labelIDs  []shared.ID
	createdAt time.Time
	updatedAt time.Time
}

// New creates a new Issue owned by projectID. phaseID is optional — nil means the
// issue lives in the project backlog and is not yet assigned to any phase.
func New(projectID shared.ID, phaseID *shared.ID, title Title) (Issue, error) {
	if projectID.IsZero() {
		return Issue{}, ErrInvalidProjectID
	}
	if !title.isValid() {
		return Issue{}, ErrInvalidTitle
	}
	now := time.Now()
	return Issue{
		id:        shared.NewID(),
		projectID: projectID,
		phaseID:   phaseID,
		issueType: IssueTypeTask,
		title:     title,
		status:    StatusOpen,
		priority:  PriorityMedium,
		labelIDs:  []shared.ID{},
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute rebuilds an Issue from persisted data.
// Bypasses constructor validation — callers must ensure data integrity.
func Reconstitute(id, projectID shared.ID, phaseID *shared.ID, issueType IssueType, title Title, spec string, status Status, priority Priority, dueDate *time.Time, labelIDs []shared.ID, createdAt, updatedAt time.Time) Issue {
	if labelIDs == nil {
		labelIDs = []shared.ID{}
	}
	return Issue{
		id:        id,
		projectID: projectID,
		phaseID:   phaseID,
		issueType: issueType,
		title:     title,
		spec:      spec,
		status:    status,
		priority:  priority,
		dueDate:   dueDate,
		labelIDs:  labelIDs,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (i Issue) ID() shared.ID        { return i.id }
func (i Issue) ProjectID() shared.ID { return i.projectID }
func (i Issue) PhaseID() *shared.ID  { return i.phaseID }
func (i Issue) Type() IssueType      { return i.issueType }
func (i Issue) Title() Title         { return i.title }
func (i Issue) Spec() string         { return i.spec }
func (i Issue) Status() Status       { return i.status }
func (i Issue) Priority() Priority   { return i.priority }
func (i Issue) DueDate() *time.Time  { return i.dueDate }
func (i Issue) LabelIDs() []shared.ID { return i.labelIDs }
func (i Issue) CreatedAt() time.Time { return i.createdAt }
func (i Issue) UpdatedAt() time.Time { return i.updatedAt }
func (i Issue) IsBacklog() bool      { return i.phaseID == nil }

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

func (i *Issue) SetType(t IssueType) {
	i.issueType = t
	i.updatedAt = time.Now()
}

func (i *Issue) SetDueDate(t *time.Time) {
	i.dueDate = t
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
