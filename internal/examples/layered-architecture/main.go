package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/slavaavr/txmng"
	"github.com/slavaavr/txmng/internal/examples/layered-architecture/repo"
	"github.com/slavaavr/txmng/internal/examples/layered-architecture/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	var connString = os.Getenv("PG_DSN")

	cfg, err := pgx.ParseConfig(connString)
	if err != nil {
		panic(fmt.Errorf("parsing config: %w", err))
	}

	db := stdlib.OpenDB(*cfg)
	defer db.Close()

	txm, dbm := txmng.New(txmng.NewSQLDB(db), txmng.WithDefaultRetrier())

	someRepo := repo.NewSomeRepo(dbm)
	someService := service.NewSomeService(txm, someRepo)

	res, err := someService.Do(ctx)
	if err != nil {
		panic(err)
	}

	log.Println(res)
}
