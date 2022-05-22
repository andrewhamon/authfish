package api_key

import "time"

type ApiKey struct {
	Id        int64      `db:"id"`
	UserId    int64      `db:"user_id"`
	Memo      string     `db:"memo"`
	Key       string     `db:"key"`
	CreatedAt time.Time  `db:"created_at"`
	LastSeen  *time.Time `db:"last_seen"`
}
