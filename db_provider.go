package txmng

import "database/sql"

//go:generate mockgen -source=./db_provider.go -destination=./db_provider_mock.go -package txmng

type DBProvider interface {
	Raw() DB
	Tx(opts Opts) (DB, error)
}

type dbProvider struct {
	raw func() DB
	tx  func(opts Opts) (DB, error)
}

func (p *dbProvider) Raw() DB                  { return p.raw() }
func (p *dbProvider) Tx(opts Opts) (DB, error) { return p.tx(opts) }

func NewDefaultSQLDBProvider(db *sql.DB) DBProvider {
	return &dbProvider{
		raw: func() DB { return newDBRawAdapter(db) },
		tx: func(opts Opts) (DB, error) {
			return db.BeginTx(opts.Ctx, &sql.TxOptions{
				Isolation: opts.Isolation.toSQL(),
				ReadOnly:  opts.ReadOnly,
			})
		},
	}
}
