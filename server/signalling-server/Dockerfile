# Build Container
FROM golang:1.23-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

# Deployment pod
FROM alpine:3.21
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=build /app/main .

EXPOSE 8080
CMD ["./main"]
