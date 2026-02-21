# syntax=docker/dockerfile:1

FROM docker.io/golang:1.25.5-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/resysched ./cmd/resysched

FROM docker.io/alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/resysched /app/resysched
EXPOSE 8080
ENTRYPOINT ["/app/resysched"]
CMD ["server"]
