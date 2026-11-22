-- +goose Up
-- El TL, los devs de una etapa y el dueño de una tarea pueden ser un agente o un
-- usuario (perfil por sesión). Ya no apuntan solo a agents: assign_stage_lead,
-- add_stage_member y assign_task comprueban que el nombre sea un agente activo o
-- un usuario activo, y change_task_status pone como dueño al actor autenticado.
-- Los nombres no cambian y ni agents ni users borran filas, así que el texto
-- sigue siendo estable, como actor_name en los historiales.
ALTER TABLE stages DROP CONSTRAINT stages_lead_fkey;
ALTER TABLE tasks DROP CONSTRAINT tasks_assignee_fkey;
ALTER TABLE stage_members DROP CONSTRAINT stage_members_agent_fkey;

-- +goose Down
-- Falla si algún usuario ya es TL, dev o dueño de una tarea.
ALTER TABLE stage_members ADD CONSTRAINT stage_members_agent_fkey FOREIGN KEY (agent) REFERENCES agents (name) ON DELETE RESTRICT;
ALTER TABLE tasks ADD CONSTRAINT tasks_assignee_fkey FOREIGN KEY (assignee) REFERENCES agents (name) ON DELETE RESTRICT;
ALTER TABLE stages ADD CONSTRAINT stages_lead_fkey FOREIGN KEY (lead) REFERENCES agents (name) ON DELETE RESTRICT;
