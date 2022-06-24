FROM golang:1.18.3-bullseye

# Set our working directory
WORKDIR /app

# Copy in required files to build the app
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd

# install our app
RUN go build -o /envoy cmd/main.go

# Environment variables
ENV ENVOY_HOST="192.168.40.11"
ENV INFLUXDB="http://192.168.10.10:9086"

# Run it
CMD ["/envoy"]