```
docker compose up -d
go run .
```

Example output
```
time=2026-01-18T00:35:09.515+03:00 level=INFO msg=Helloe trace_id=9124c8a07ad79a455de036dd1b45f18e
time=2026-01-18T00:35:09.520+03:00 level=INFO msg=Connect host=localhost port=35444 database=postgres duration=4.672727ms pid=158 trace_id=9124c8a07ad79a455de036dd1b45f18e
time=2026-01-18T00:35:09.520+03:00 level=DEBUG msg=Acquire duration=4.741612ms pid=158 trace_id=9124c8a07ad79a455de036dd1b45f18e
time=2026-01-18T00:35:09.520+03:00 level=DEBUG msg=Release pid=158 trace_id=00000000000000000000000000000000
time=2026-01-18T00:35:09.520+03:00 level=DEBUG msg=Acquire duration=6.478µs pid=158 trace_id=9124c8a07ad79a455de036dd1b45f18e
time=2026-01-18T00:35:09.520+03:00 level=INFO msg=Prepare duration=478.346µs alreadyPrepared=false pid=158 name=stmtcache_7bd44f3460adcdc12596a2941974902e260f35b0115ebc93 sql="select id, title from chat_common where id=$1 or id=$2" trace_id=9124c8a07ad79a455de036dd1b45f18e
time=2026-01-18T00:35:09.521+03:00 level=INFO msg=Query sql="select id, title from chat_common where id=$1 or id=$2" args="[1 2]" duration=1.272632ms commandTag="SELECT 2" pid=158 trace_id=9124c8a07ad79a455de036dd1b45f18e
time=2026-01-18T00:35:09.521+03:00 level=DEBUG msg=Release pid=158 trace_id=00000000000000000000000000000000
Results:
[]main.Dto{
  main.Dto{
    Ide: 1,
    Titled: "Chat of souls",
  },
  main.Dto{
    Ide: 2,
    Titled: "Not a chat",
  },
}
```

http://localhost:46686/jaeger/trace/9124c8a07ad79a455de036dd1b45f18e