-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email,hashed_password,is_chirpy_red)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    FALSE
)
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserEmailAndPassword :one
UPDATE users SET email = $1, hashed_password = $2 WHERE id = $3
RETURNING *;


-- name: UpdateChirpyRedMembership :exec
UPDATE users SET is_chirpy_red = True WHERE id = $1;