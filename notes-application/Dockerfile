FROM golang:1.26
WORKDIR /app

# Flaw: secret baked into an image layer — visible in image history,
# present in every container started from this image.
ENV ADMIN_API_KEY=sk-live-7f3a9c2e41d8b6f0

COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o server .

# Flaw: runs as root, no USER directive.
CMD ["./server"]
