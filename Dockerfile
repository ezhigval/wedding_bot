FROM golang:1.24-alpine3.22 AS builder

RUN apk add --no-cache gcc musl-dev nodejs npm

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY webapp-react/package.json webapp-react/package-lock.json ./webapp-react/
RUN cd webapp-react && npm ci

COPY . .

RUN cd webapp-react && npm run build
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk --no-cache add ca-certificates \
	&& addgroup -S app \
	&& adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/server ./server
COPY --from=builder /app/webapp ./webapp
COPY --from=builder /app/res ./res

RUN mkdir -p /app/data && chown -R app:app /app

USER app

ENV PORT=10000

EXPOSE 10000

CMD ["./server"]
