FROM golang:alpine AS builder

WORKDIR /src

COPY ./server/ ./

RUN mkdir -p /app && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/srv .

FROM alpine:3.22

RUN apk add --no-cache \
    bash \
    openvpn \
    iptables \
    ip6tables \
    iproute2 \
    ca-certificates \
    tini \
    curl \
    su-exec

RUN mkdir -p \
    /vpn \
    /app/web \
    /app/conf.d \
    /usr/local/bin

COPY --from=builder /app/srv /vpn/srv

COPY ./conf/config.ovpn /vpn/config.ovpn
COPY ./web/ /app/web/

COPY ./scripts/openvpn.sh /usr/local/bin/openvpn.sh
COPY ./scripts/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod 755 \
    /vpn/srv \
    /usr/local/bin/openvpn.sh \
    /usr/local/bin/entrypoint.sh

WORKDIR /vpn

VOLUME ["/app/conf.d"]

EXPOSE 8080 8181 8989 7878 9696

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]

