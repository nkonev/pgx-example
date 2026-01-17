```
docker compose up -d
go run .
```

Example output
```
time=2026-01-18T00:03:34.868+03:00 level=INFO msg=Helloe trace_id=8e3357c4214056b7c4154cb1b2574240
time=2026-01-18T00:03:34.874+03:00 level=INFO msg=Query sql="select id, title from chat_common where id=$1 or id=$2" args="[1 2]" time=1.223024ms commandTag="SELECT 2" pid=74 trace_id=8e3357c4214056b7c4154cb1b2574240
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

http://localhost:46686/jaeger/trace/8e3357c4214056b7c4154cb1b2574240