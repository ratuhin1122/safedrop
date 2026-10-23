FROM golang:1.25-bookworm AS dependencies

RUN apt-get update \
    && apt-get install -y --no-install-recommends libvips-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*

FROM dependencies AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -trimpath -o /out/api ./cmd/api \
    && go build -trimpath -o /out/worker ./cmd/worker \
    && go build -trimpath -o /out/migrate ./cmd/migrate

FROM build AS test
RUN go test ./...

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libvips42 curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 safedrop \
    && useradd --uid 10001 --gid safedrop --no-create-home safedrop \
    && mkdir -p /data/incoming /data/final \
    && chown -R safedrop:safedrop /data \
    && chmod 700 /data /data/incoming /data/final
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/worker /usr/local/bin/worker
COPY --from=build /out/migrate /usr/local/bin/migrate
WORKDIR /data
USER 10001:10001
EXPOSE 8080
CMD ["/usr/local/bin/api"]
