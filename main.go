package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"
	"github.com/sanity-io/litter"
)

func main() {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://postgres:postgresqlPassword@localhost:5432/chat?sslmode=disable&application_name=cqrs-app")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	type Dto struct {
		Ide    int64  `db:"id"`
		Titled string `db:"title"`
	}

	mapped2, err := pgxutil.Select(ctx, conn, "select id, title from chat_common where id=$1", []any{1}, pgx.RowToStructByName[Dto])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
		os.Exit(1)
	}
	litter.Dump(mapped2)
}
