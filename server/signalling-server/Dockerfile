# Build pod - Used for building the microservice from source
FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /microservice

# Deployment Pod - Used for the deployment of the MS
FROM alpine:latest
WORKDIR /root/src

# Install necessary tools: curl and bash
RUN apk add --no-cache curl bash && \
    curl -o /tmp/consul.zip https://releases.hashicorp.com/consul/1.9.4/consul_1.9.4_linux_amd64.zip && \
    unzip /tmp/consul.zip -d /bin && \
    rm /tmp/consul.zip

# Copy Consul service configuration
COPY consul-service.json /etc/consul.d/service.json

# Copy the built microservice binary from the build stage
COPY --from=build /microservice .

# Start Consul agent and the microservice
CMD ["consul", "agent", "-data-dir=/consul/data", "-config-dir=/etc/consul.d", "-client=0.0.0.0", "&", "./microservice"]
