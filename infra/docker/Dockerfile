# ----- Builder -----
FROM golang:1.26.10-bookworm AS builder

# Install gRPC & GraphQL tool
RUN go install github.com/bufbuild/buf/cmd/buf@v1.72.0 && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2 && \
    go install github.com/99designs/gqlgen@v0.17.94
ENV PATH="/root/go/bin:${PATH}"

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy sources
COPY . .

# Static binary, stamped with the image version
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/financecore ./cmd

# -------------------

# ----- Runtime -----
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=builder /out/financecore ./financecore

ENTRYPOINT ["/app/financecore"]