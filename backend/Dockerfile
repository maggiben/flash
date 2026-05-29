FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ ./
COPY specs/ /app/specs/
COPY seeds/ /app/seeds/

RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM golang:1.23-alpine AS test

WORKDIR /app

RUN apk add --no-cache git

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ ./
COPY specs/ /app/specs/
COPY seeds/ /app/seeds/

CMD ["go", "test", "-v", "-count=1", "./test/integration/..."]

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /server /app/server
COPY backend/migrations /app/migrations
COPY seeds /app/seeds
COPY specs /app/specs

ENV PORT=8080
EXPOSE 8080

CMD ["/app/server"]
