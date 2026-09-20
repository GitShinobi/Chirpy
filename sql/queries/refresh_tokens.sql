-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at,revoked_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3,
    NULL
)
RETURNING *;

-- name: DeleteRefreshTokens :exec
DELETE FROM refresh_tokens;


-- name: AddrevokedAt :exec
UPDATE refresh_tokens 
SET revoked_at = NOW(), updated_at = NOW() 
WHERE user_id = $1;


-- name: GetUserFromRefreshToken :one
SELECT * FROM refresh_tokens WHERE token=$1;