
Build go with `go build .`

Run locally with `source env.sh && ./go-azure-ad`

Build and run docker with 
```
docker build -t ryandawsonuk/go-azure-ad:test .
docker run -p 8080:8080 ryandawsonuk/go-azure-ad:test --env-file env.list
```