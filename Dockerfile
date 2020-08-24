FROM golang

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

ENV CLIENT_ID="kubeflow-oidc-authservice"
ENV AUTH_URL="http://188.166.138.216/dex/auth"
ENV TOKEN_URL="http://188.166.138.216/dex/token"
ENV OIDC_SCOPES="profile email groups"
ENV REDIRECT_URL="http://localhost:8000/seldon-deploy/auth/callback"
ENV RESOURCE_URI="https://graph.windows.net"
ENV CLIENT_SECRET="pUBnBOY80SnXgjibTYM9ZWNzY2xreNGQok"

EXPOSE 8000
ENTRYPOINT ["/app/go-azure-ad"]