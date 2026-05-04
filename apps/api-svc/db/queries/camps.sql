-- name: GetCamp :one
SELECT * FROM camps WHERE id = $1;

-- name: ListCamps :many
SELECT * FROM camps
WHERE (sqlc.narg('sport')::text IS NULL OR sport = sqlc.narg('sport'))
ORDER BY start_date ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CreateCamp :one
INSERT INTO camps (name, sport, location, capacity, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateCamp :one
UPDATE camps
SET name = $2,
    sport = $3,
    location = $4,
    capacity = $5,
    start_date = $6,
    end_date = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCamp :exec
DELETE FROM camps WHERE id = $1;
