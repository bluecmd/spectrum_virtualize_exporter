FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags=-s -o main .

FROM scratch
WORKDIR /opt/spectrum_virtualize_exporter

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/main .

EXPOSE 9747
USER 65534
ENTRYPOINT ["./main"]
