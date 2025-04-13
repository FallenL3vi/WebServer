-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeletePosts :exec
DELETE FROM posts;

-- name: GetPosts :many
SELECT * FROM posts
ORDER BY created_at ASC;

-- name: GetPost :one
SELECT * FROM posts
WHERE $1 = id;

-- name: DeletePost :execresult
DELETE FROM posts
WHERE $1 = id AND user_id = $2;