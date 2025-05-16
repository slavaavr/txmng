package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/slavaavr/txmng"
	"github.com/slavaavr/txmng/examples/layered-architecture/repos"
	"github.com/slavaavr/txmng/examples/layered-architecture/services"
)

func main() {
	var connectString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost", "5432", "user", "12345678", "test")

	db, err := sql.Open("postgres", connectString)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dbProvider := txmng.NewDefaultSQLDBProvider(db)
	txm, dbm := txmng.New(dbProvider)

	repo := repos.NewSomeRepo(dbm)
	service := services.NewSomeService(txm, repo)

	res, err := service.Do(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("yay, we've got the answer: %d\n", res)
}
