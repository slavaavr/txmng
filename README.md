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

dbProvider := txmng.NewPGXProvider(db)
txm, dbm := txmng.New(dbProvider, txmng.WithDefaultRetrier())

r := repo.New(dbm)
s := service.New(txm, r)
s.Do()

// service.go
func (s *Service) Do() error {
    txOpts := txmng.TxOpts{
        Ctx:       ctx,
        Isolation: txmng.LevelDefault,
        ReadOnly:  false,
        Ext:       nil,
    }
	
    scanner, err := s.txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
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
        e1 *SomeEntity1
        e2 *SomeEntity2
    )
	
    if err := scanner.Scan(&e1, &e2); err != nil {
        return fmt.Errorf("scan: %w", err)
    }
	
    return s.process(e1, e2)
}

// repo.go
func (r *Repo) Do1(ctx txmng.Context, params Params) (*SomeEntity1, error) {
    db, err := r.dbm.GetDB(ctx)
    if err != nil {
        return nil, err	
    }
	
    return r.query(db, params)
}

func (r *Repo) Do2(ctx txmng.Context, params Params) (*SomeEntity2, error) {
    db := r.dbm.MustGetDB(ctx)
    return r.query(db, params)
}
```

### Notes
For more details, see the [examples](https://github.com/slavaavr/txmng/tree/master/internal/examples) folder.

[ci-badge]:      https://github.com/slavaavr/txmng/actions/workflows/main.yaml/badge.svg
[ci-runs]:       https://github.com/slavaavr/txmng/actions