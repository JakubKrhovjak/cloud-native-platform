FROM golang:1.25-alpine

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /build

# Copy api module
COPY --chown=appuser:appuser api/ ./api/

# Copy student-service module
COPY --chown=appuser:appuser student-service/ ./student-service/

# Create go.work for mono repo
RUN printf 'go 1.25.4\n\nuse (\n\t./api\n\t./student-service\n)' > go.work

# Download dependencies
WORKDIR /build/student-service
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Move to runtime directory
WORKDIR /home/appuser
RUN mv /build/student-service/server . && \
    mv /build/student-service/configs ./configs

USER appuser

EXPOSE 8080

CMD ["./server"]
