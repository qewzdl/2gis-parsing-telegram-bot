FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o parser ./cmd/bot

# ---- runtime image ----
FROM alpine:3.19

RUN apk add --no-cache sqlite-libs ca-certificates tzdata
ENV TZ=Asia/Almaty

WORKDIR /app
COPY --from=builder /app/parser .

RUN mkdir -p data exports

CMD ["./parser"]
