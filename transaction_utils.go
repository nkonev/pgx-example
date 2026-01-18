package main

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

type Transactioner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func TransactWithResult[T any](ctx context.Context, db Transactioner, txFunc func(Querier) (T, error)) (ret T, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback(ctx) // err is non-nil; don't change it
		} else {
			err = tx.Commit(ctx) // err is nil; if Commit returns error update err
		}
	}()
	ret, err = txFunc(tx)
	return ret, err
}

func Transact(ctx context.Context, db Transactioner, txFunc func(Querier) error) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback(ctx) // err is non-nil; don't change it
		} else {
			err = tx.Commit(ctx) // err is nil; if Commit returns error update err
		}
	}()
	err = txFunc(tx)
	return err
}
