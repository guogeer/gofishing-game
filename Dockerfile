FROM atomhub.openatom.cn/amd64/golang:1.21.1-bookworm AS gobuilder

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum .
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o ./bin/hall_server ./hall/*.go
RUN go build -ldflags="-s -w" -o ./bin/cache_server ./cache/*.go
RUN go build -ldflags="-s -w" -o ./bin/games_server ./games/*.go
RUN go build -ldflags="-s -w" -o ./bin/login_server ./login/*.go
RUN go install github.com/guogeer/quasar/v2/...@v2.0.5

FROM atomhub.openatom.cn/amd64/node:20.6.1-bookworm AS nodebuilder
WORKDIR /build
COPY tests/web/package.json .
RUN npm install
COPY tests/web .
RUN npm run build

FROM atomhub.openatom.cn/amd64/debian:12.1-slim

WORKDIR /app
COPY docker/config.yaml config.yaml
COPY tests tests
COPY --from=gobuilder /build/bin .
COPY --from=gobuilder /go/bin/gateway gateway_server
COPY --from=gobuilder /go/bin/router router_server
COPY --from=nodebuilder /build/dist www
