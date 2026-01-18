package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/exaring/otelpgx"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	pgxSlog "github.com/mcosta74/pgx-slog"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"

	"github.com/jackc/pgx/v5/multitracer"
	"github.com/sanity-io/litter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	trmcontext "github.com/avito-tech/go-transaction-manager/trm/v2/context"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

func New(ctx context.Context, dsn string, provider trace.TracerProvider, slogger *slog.Logger) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(err)
	}

	// FYI: NewLoggerAdapter: https://github.com/mcosta74/pgx-slog
	adapterLogger := pgxSlog.NewLogger(slogger)

	// https://github.com/jackc/pgx/discussions/1677#discussioncomment-12253699
	m := multitracer.New(
		otelpgx.NewTracer(otelpgx.WithTracerProvider(provider)),
		&tracelog.TraceLog{
			Logger:   adapterLogger,
			LogLevel: tracelog.LogLevelTrace,
			Config: &tracelog.TraceLogConfig{
				TimeKey: "duration",
			},
		},
	)

	config.ConnConfig.Tracer = m

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		panic(err)
	}

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	return pool
}

const LogFieldTraceId = "trace_id"

func GetTraceId(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	tr := sc.TraceID()
	return tr.String()
}

type TracingContextHandler struct {
	slog.Handler
}

func (h *TracingContextHandler) Handle(ctx context.Context, r slog.Record) error {
	traceId := GetTraceId(ctx)
	if traceId != "" {
		r.AddAttrs(slog.String(LogFieldTraceId, traceId))
	}

	return h.Handler.Handle(ctx, r)
}

func main() {
	ctx := context.Background()

	h := &TracingContextHandler{slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})}
	sl := slog.New(h)

	traceExporterConn, err := grpc.DialContext(ctx, "localhost:44317", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		panic(err)
	}
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(traceExporterConn))

	defer exporter.Shutdown(ctx)
	defer traceExporterConn.Close()

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("my-app"),
	)
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(resources),
	)
	otel.SetTracerProvider(tp)

	// register jaeger propagator
	otel.SetTextMapPropagator(jaegerPropagator.Jaeger{})

	defer tp.Shutdown(ctx)

	tr := tp.Tracer("my.tracer")
	ctx, span := tr.Start(ctx, "my.span")
	defer span.End()

	sl.InfoContext(ctx, "Helloe")

	pool := New(ctx, "postgres://postgres:postgresqlPassword@localhost:35444/postgres?sslmode=disable&application_name=pgx-trace-app", tp, sl)
	defer pool.Close()

	type Dto struct {
		Ide    int64  `db:"id"`
		Titled string `db:"title"`
	}

	dts := []Dto{}

	err = pgxscan.Select(ctx, pool, &dts, "select id, title from chat_common where id=$1 or id=$2", 1, 2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
		os.Exit(1)
	}
	sl.InfoContext(ctx, "Results:")
	litter.Dump(dts)

	// compatibility level
	stdDb := stdlib.OpenDBFromPool(pool)

	dtsComp := []Dto{}
	err = sqlscan.Select(ctx, stdDb, &dtsComp, "select title, id from chat_common where id = $1", 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
		os.Exit(1)
	}

	sl.InfoContext(ctx, "ResultsComp:")
	litter.Dump(dtsComp)

	g := trmsqlx.DefaultCtxGetter
	dm := trmcontext.DefaultManager

	trManager := manager.Must(trmsqlx.NewDefaultFactory(pool))

	err = trManager.Do(ctx, func(ctx context.Context) error {

		trq := g.DefaultTrOrDB(ctx, pool)

		_, err = trq.Exec(ctx, "update chat_common set title = 'new one ' || now()::text where id = $1", 1)

		return err
	})

	rows, _ := g.DefaultTrOrDB(ctx, pool).Query(ctx, "select id, title from chat_common where id = $1", 1)

	gotten, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[Dto])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
		os.Exit(1)
	}
	sl.InfoContext(ctx, "gotten:")
	litter.Dump(gotten)

	g2 := Dto{}
	err = pgxscan.Get(ctx, g.DefaultTrOrDB(ctx, pool), &g2, "SELECT id, title FROM chat_common WHERE id = $1", 2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
		os.Exit(1)
	}
	sl.InfoContext(ctx, "gotten2:")
	litter.Dump(g2)

	err = trManager.Do(ctx, func(ctx context.Context) error {
		tx := g.DefaultTrOrDB(ctx, pool)

		_, err = tx.Exec(ctx, "update chat_common set title = 'new two ' || now()::text where id = $1", 2)
		_, err = tx.Exec(ctx, "update chat_common set title = 'new two second ' || now()::text where id = $1", 2)
		return errors.New("Not now")
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tx failed1: %v\n", err)
	}

	err = trManager.Do(ctx, func(ctx context.Context) error {
		tx := dm.Default(ctx).Transaction().(trmsqlx.Tr)

		_, err = tx.Exec(ctx, "update chat_common set title = 'new two ' || now()::text where id = $1", 2)
		_, err = tx.Exec(ctx, "update chat_common set title = 'new two second ' || now()::text where id = $1", 2)
		panic("The panic!")
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tx failed2: %v\n", err)
	}

	g3 := Dto{}
	err = pgxscan.Get(ctx, g.DefaultTrOrDB(ctx, pool), &g3, "SELECT id, title FROM chat_common WHERE id = $1", 2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
		os.Exit(1)
	}
	sl.InfoContext(ctx, "gotten3:")
	litter.Dump(g3)
}
