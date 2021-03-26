FROM quay.io/prometheus/golang-builder:1.16.2-base as builder

WORKDIR /build

COPY . .
RUN go get -v -t -d ./...
RUN CGO_ENABLED=0 go build -o main .

FROM scratch
WORKDIR /opt/spectrum_virtualize_exporter

COPY --from=builder /build/main .

EXPOSE 9747
CMD ["./main", "-auth-file", "~/spectrum-monitor.yaml", "-extra-ca-cert", "~/tls.crt"]
