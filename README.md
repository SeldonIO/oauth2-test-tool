
https://github.com/jlubawy/go-azure-ad is an Active Directory OAuth example in Go. It contains a UI for logging in.

Build go with `go build .`

Run locally with `source env.sh && ./go-azure-ad`

Build and run docker with 
```
docker build -t ryandawsonuk/go-azure-ad:test .
docker run --env-file env.list -p 8000:8000 ryandawsonuk/go-azure-ad:test
```

Side Note On Running Against Dex
```
kubectl get configmap dex -n auth -o jsonpath='{.data.config\.yaml}' > dex-config.yaml
kubectl create cm dex --from-file=config.yaml=dex-config.yaml --dry-run -oyaml | kubectl apply -f - --namespace=auth
kubectl delete pods -n auth -l app=dex
```