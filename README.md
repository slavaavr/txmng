# txmng

`txmng` is a library for managing transactions at the service layer in a layered architecture.

[![Build Status][ci-badge]][ci-runs]

### Installation

```sh
$ go get -u github.com/slavaavr/txmng
```

### Quick start
```go
// main.go
var db *pgxpool.Pool = initDB()

txm, dbm := txmng.New(
	txmng.NewPGXProvider(db),
	txmng.WithDefaultRetrier(),
)

repo := repo.New(dbm)
service := service.New(txm, repo)
service.Do() // run business logic

// service.go
func (s *Service) Do() error {
    opts := txmng.TxOpts{
        Ctx:       ctx,
        Isolation: txmng.LevelDefault,
        ReadOnly:  false,
        Ext:       nil,
    }
	
    scanner, err := s.txm.RunTx(opts, func(ctx txmng.Context) (txmng.Scanner, error) {
        e1, err := s.repo.Do1(ctx, params)
        if err != nil {
            return nil, fmt.Errorf("exec Do1: %w", err)
        }

        e2, err := s.repo.Do2(ctx, params)
        if err != nil {
            return nil, fmt.Errorf("exec Do2: %w", err)
        }
		
        return txmng.Values(e1, e2), nil
    })
	if err != nil {
	    return fmt.Errorf("exec tx: %w", err)	
    }   
	
    var (
        e1 *Entity1
        e2 *Entity2
    )
	
    if err := scanner.Scan(&e1, &e2); err != nil {
        return fmt.Errorf("scan: %w", err)
    }
	
    return s.process(e1, e2)
}

// repo.go
func (r *Repo) Do1(ctx txmng.Context, params Params) (*Entity1, error) {
    db, err := r.dbm.GetDB(ctx)
    if err != nil {
        return nil, err	
    }
	
    return r.query(db, params)
}

func (r *Repo) Do2(ctx txmng.Context, params Params) (*Entity2, error) {
    db := r.dbm.MustGetDB(ctx)
    return r.query(db, params)
}
```

### Notes
For more details, see the [examples](https://github.com/slavaavr/txmng/tree/master/internal/examples) folder.

[ci-badge]:      https://github.com/slavaavr/txmng/actions/workflows/main.yaml/badge.svg
[ci-runs]:       https://github.com/slavaavr/txmng/actions
