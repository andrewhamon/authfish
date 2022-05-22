package context

import (
	"net/url"

	"github.com/jmoiron/sqlx"
)

type AppContext struct {
	Db      *sqlx.DB
	BaseUrl *url.URL
}
