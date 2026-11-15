# Build and runtime container for the Todo API

FROM golang:1.26

LABEL maintainer="devfest-surabaya"
LABEL description="Todo Microservice Backend"

WORKDIR /app

ENV TODO_ADMIN_KEY=dev-test-admin-secret-key-3b9e4a1
ENV PORT=8080

# Cache module layers
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile statically
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/todo-service .

EXPOSE 8080

ENTRYPOINT ["/app/todo-service"]
