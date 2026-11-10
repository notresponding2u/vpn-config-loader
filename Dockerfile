FROM golang:latest as builder

WORKDIR /usr/local/app


COPY ./server /app/server

RUN cd /app/server && go build -o /app/srv ./ 

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        openvpn \
        iproute2 \
        iptables \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*



# OpenVPN configuration directory
RUN mkdir -p /etc/openvpn /app/conf

VOLUME ["/app/conf.d", "/vpn"]

COPY ./conf/config.ovpn /app/conf

COPY ./auth.txt /app

COPY --from=builder /app/srv /app/srv

WORKDIR /app

ENTRYPOINT ["/app/srv"]
