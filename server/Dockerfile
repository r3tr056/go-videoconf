
# Build Container
FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main .

# Deployment pod
FROM alpine:latest
WORKDIR /root/src
COPY --from=build /app/main .

EXPOSE 8080
CMD ["./main"]
