FROM docker.io/golang:1.22-alpine AS builder
RUN apk add --no-cache ca-certificates tzdata git
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build app
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/resysched ./...

# Build resy-cli and normalize binary name to /out/resy
RUN go install github.com/lgrees/resy-cli@latest &&     if [ -f /go/bin/resy-cli ]; then cp /go/bin/resy-cli /out/resy; fi &&     if [ -f /go/bin/resy ]; then cp /go/bin/resy /out/resy; fi

FROM docker.io/alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /out/resysched /usr/local/bin/resysched
COPY --from=builder /out/resy /usr/local/bin/resy

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/resysched"]
CMD ["server","--migrate=true"]
