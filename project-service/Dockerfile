FROM golang:1.25-alpine

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /home/appuser

COPY --chown=appuser:appuser go.mod go.sum ./
RUN go mod download

COPY --chown=appuser:appuser . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

USER appuser

EXPOSE 8081

CMD ["./server"]
