package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

var dbInstance *pgxpool.Pool
var lgr *slog.Logger

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func shutdown() {
	if dbInstance != nil {
		dbInstance.Close()
	}
}

func setup() {
	lgr = slog.New(slog.NewTextHandler(os.Stdout, nil))

	config, err := pgxpool.ParseConfig("postgres://postgres:postgresqlPassword@localhost:35444/postgres?sslmode=disable&application_name=pgx-trace-app")
	if err != nil {
		panic(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}

	dbInstance = pool

	if err != nil {
		panic(fmt.Sprintf("Error during getting db connection for test: %v", err))
	} else {
		RecreateDb(dbInstance)
	}
}

func RecreateDb(q Querier) {
	_, err := q.Exec(context.Background(), `
		-- test
		drop table if exists t1;
		drop table if exists t2;
		drop table if exists tr1;
		drop table if exists tr2;
	`)
	lgr.Warn("Recreating database")
	if err != nil {
		panic(fmt.Sprintf("Error during dropping db: %v", err))
	}
}

func TestTransactionPositive(t *testing.T) {
	ctx := context.Background()
	err := Transact(ctx, dbInstance, func(tx Querier) error {
		if _, err := tx.Exec(ctx, "CREATE TABLE t1(a text UNIQUE)"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "insert into t1(a) VALUES ('lorem')"); err != nil {
			return err
		}
		return nil
	})
	assert.Nil(t, err)

	row := dbInstance.QueryRow(ctx, "SELECT a FROM t1")
	var a string
	err = row.Scan(&a)
	assert.Nil(t, err)
	assert.Equal(t, "lorem", a)
}

func TestTransactionNegative(t *testing.T) {
	ctx := context.Background()

	_, err := dbInstance.Exec(ctx, "CREATE TABLE t2(a text UNIQUE)")
	assert.Nil(t, err)

	err = Transact(ctx, dbInstance, func(tx Querier) error {
		if _, err := tx.Exec(ctx, "insert into t2(a) VALUES ('lorem')"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "insert into t2(a) VALUES ('lorem')"); err != nil {
			return err
		}
		return nil
	})
	assert.NotNil(t, err)

	row := dbInstance.QueryRow(ctx, "SELECT a FROM t2")
	var a string
	err = row.Scan(&a)
	assert.NotNil(t, err)
	s := err.Error()
	assert.Equal(t, `no rows in result set`, s)
}

func TestTransactionWithResultPositive(t *testing.T) {
	ctx := context.Background()

	id, err := TransactWithResult(ctx, dbInstance, func(tx Querier) (int64, error) {
		if _, err := tx.Exec(ctx, "CREATE TABLE tr1(id BIGSERIAL PRIMARY KEY, a text UNIQUE)"); err != nil {
			return 0, err
		}
		res := tx.QueryRow(ctx, `INSERT INTO tr1(a) VALUES ('lorem') RETURNING id`)
		var id int64
		if err := res.Scan(&id); err != nil {
			lgr.Error(fmt.Sprintf("Error during getting chat id %v", err))
			return 0, err
		}

		return id, nil
	})
	assert.Nil(t, err)

	assert.True(t, id != 0)

	row := dbInstance.QueryRow(ctx, "SELECT a FROM tr1 WHERE id = $1", id)
	var a string
	err = row.Scan(&a)
	assert.Nil(t, err)
	assert.Equal(t, "lorem", a)
}

func TestTransactionWithResultNegative(t *testing.T) {
	ctx := context.Background()

	_, err := dbInstance.Exec(ctx, "CREATE TABLE tr2(id BIGSERIAL PRIMARY KEY, a text UNIQUE)")
	assert.Nil(t, err)

	idRaw, err := TransactWithResult(ctx, dbInstance, func(tx Querier) (int64, error) {
		res := tx.QueryRow(ctx, `INSERT INTO tr2(a) VALUES ('lorem') RETURNING id`)
		var id int64
		if err := res.Scan(&id); err != nil {
			lgr.Error(fmt.Sprintf("Error during getting chat id %v", err))
			return 0, err
		}
		if _, err := tx.Exec(ctx, "insert into tr2(a) VALUES ('lorem')"); err != nil {
			return 0, err
		}

		return id, nil
	})
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), idRaw)

	row := dbInstance.QueryRow(ctx, "SELECT a FROM tr2")
	var a string
	err = row.Scan(&a)
	assert.NotNil(t, err)
	s := err.Error()
	assert.Equal(t, `no rows in result set`, s)
}
