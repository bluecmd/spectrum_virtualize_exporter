# Cross-compiles on the build platform, so a multi-arch build needs no
# emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags=-s -o main .

FROM scratch
WORKDIR /opt/spectrum_virtualize_exporter

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/main .

EXPOSE 9747
USER 65534
ENTRYPOINT ["./main"]
