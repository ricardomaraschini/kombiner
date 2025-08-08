FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum .
RUN go mod download
COPY . .
RUN make build

FROM fedora:43
COPY --from=builder /app/_output/bin/ /usr/local/bin/
