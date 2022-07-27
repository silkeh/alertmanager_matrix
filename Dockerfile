FROM --platform=$BUILDPLATFORM golang:alpine AS build
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ENV GOPATH="/build/.go"
COPY [".", "/build"]
RUN cd /build && CGO_ENABLED=0 go build -ldflags="-s -w" ./cmd/alertmanager_matrix

FROM scratch
COPY --from=build ["/etc/ssl/cert.pem", "/etc/ssl/certs/ca-certificates.crt"] 
COPY --from=build ["/build/alertmanager_matrix", "/usr/local/bin/"]
ENTRYPOINT ["/usr/local/bin/alertmanager_matrix"]
