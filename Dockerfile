# -------- build stage --------
FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o tfsuit ./cmd/tfsuit

# -------- final image --------
FROM scratch
LABEL org.opencontainers.image.source="https://github.com/josdagaro/tfsuit"
COPY --from=builder /src/tfsuit /usr/local/bin/tfsuit
ENTRYPOINT ["tfsuit"]
