-- name: CreatePushSubscription :one
INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth)
VALUES (?, ?, ?, ?, ?)
RETURNING id, user_id, endpoint, p256dh, auth, created_at, last_seen_at;

-- name: GetPushSubscriptionByEndpoint :one
SELECT id, user_id, endpoint, p256dh, auth, created_at, last_seen_at
FROM push_subscriptions
WHERE endpoint = ?;

-- name: UpdatePushSubscription :one
UPDATE push_subscriptions
SET p256dh = ?, auth = ?, last_seen_at = CURRENT_TIMESTAMP
WHERE endpoint = ?
RETURNING id, user_id, endpoint, p256dh, auth, created_at, last_seen_at;

-- name: ListAllPushSubscriptions :many
SELECT id, user_id, endpoint, p256dh, auth, created_at, last_seen_at
FROM push_subscriptions;

-- name: CountPushSubscriptionsByUser :one
SELECT COUNT(*) FROM push_subscriptions WHERE user_id = ?;

-- name: DeletePushSubscriptionByEndpoint :exec
DELETE FROM push_subscriptions WHERE endpoint = ?;
