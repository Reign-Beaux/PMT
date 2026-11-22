-- name: HoldOf :many
-- Quién tiene tomada la etapa o el bug que contiene a un elemento (pmt_hold_of).
-- Sin filas si no pertenece a ninguno; ID nulo y nombre vacío si nadie lo tiene.
SELECT h.holder_kind::text AS holder_kind,
       COALESCE(h.taken_by, '00000000-0000-0000-0000-000000000000'::uuid)::uuid AS taken_by,
       COALESCE(h.taken_by_name, '')::text AS taken_by_name
FROM pmt_hold_of(@kind::text, @item_id::uuid) h;
