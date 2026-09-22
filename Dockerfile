FROM golang:1.27-alpine AS builder

LABEL maintainer="Candro Aleandro <admin@tanet.eu.org> (https://tanet.eu.org/)"

# Move to working directory (/build).
WORKDIR /build

# Copy and download dependency using go mod.
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container.
COPY . .

# Set necessary environment variables needed for our image and build the API server.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o apiserver .

FROM scratch

# Copy binary and config files from /build to root folder of scratch container.
COPY --from=builder ["/build/apiserver", "/build/.env", "/"]

# Probes (see pkg/routes/health_routes.go). The orchestrator, not the app,
# decides restart/unready based on these.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["/apiserver", "-healthcheck"]

# Command to run when starting the container.
ENTRYPOINT ["/apiserver"]