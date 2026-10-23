# ----- Builder -----
FROM golang:1.26.10-bookworm AS builder

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build migrate CLI
RUN CGO_ENABLED=0 GOOS=linux go build -tags postgres -trimpath -ldflags="-s -w" \
    -o /out/migrate github.com/golang-migrate/migrate/v4/cmd/migrate

# --------------------

# ----- Runtime -----
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /out/migrate /usr/local/bin/migrate
COPY migrations /migrations

ENTRYPOINT ["/usr/local/bin/migrate", "-path", "/migrations"]
