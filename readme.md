```
docker compose up -d
go run .
```

Example output
```
time=2026-01-18T00:18:38.853+03:00 level=INFO msg=Helloe trace_id=adfc85149298bc36f03f65448dc3a632
time=2026-01-18T00:18:38.859+03:00 level=INFO msg=Query sql="select id, title from chat_common where id=$1 or id=$2" args="[1 2]" duration=1.363543ms commandTag="SELECT 2" pid=119 trace_id=adfc85149298bc36f03f65448dc3a632
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

http://localhost:46686/jaeger/trace/adfc85149298bc36f03f65448dc3a632