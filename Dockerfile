FROM golang:1.22-alpine AS build

WORKDIR /app

# Install required build packages
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Cache modules first
COPY go.mod ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build statically linked binary for linux/amd64
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -o server .

# --- Runtime image ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates \
  && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=build /app/server /app/server
# Copy OpenAPI for reference (matches .choreo/endpoints.yaml path)
COPY docs/openapi.yaml /app/docs/openapi.yaml

EXPOSE 8090
ENV PORT=8090

USER 10001

ENTRYPOINT ["/app/server"]


