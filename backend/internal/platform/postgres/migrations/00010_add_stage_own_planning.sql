-- +goose Up
ALTER TABLE stages
    ADD COLUMN add_reason TEXT CHECK (add_reason <> ''),
    ADD COLUMN add_requested_by TEXT CHECK (add_requested_by <> ''),
    ADD CONSTRAINT stages_own_planning_check
        CHECK ((add_reason IS NULL) = (add_requested_by IS NULL));

-- +goose Down
ALTER TABLE stages
    DROP CONSTRAINT stages_own_planning_check,
    DROP COLUMN add_requested_by,
    DROP COLUMN add_reason;
