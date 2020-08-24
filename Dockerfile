FROM golang

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

ENV AZURE_AD_CLIENT_ID="kubeflow-oidc-authservice"
ENV AUTH_URL="http://188.166.138.216/dex/auth"
ENV TOKEN_URL="http://188.166.138.216/dex/token"
ENV OIDC_SCOPES="profile email groups"

EXPOSE 8080
ENTRYPOINT ["/app/go-azure-ad"]