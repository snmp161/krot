# Окружение для интеграционных тестов krot (SPEC 8.3): systemd как PID 1, iproute2,
# wireguard-tools и AmneziaWG (tools + userspace amneziawg-go), собранные из закреплённых тегов.
# Запуск — scripts/integration-docker.sh (контейнер со своим сетевым namespace).

FROM golang:1.27.2-bookworm AS awg
ARG AWG_TOOLS_TAG=v3.1.20260812
ARG AWG_GO_TAG=v3.1.20260828
RUN git clone -q -c advice.detachedHead=false --depth 1 --branch "$AWG_TOOLS_TAG" \
        https://github.com/amnezia-vpn/amneziawg-tools /src/tools \
    && make -s -C /src/tools/src \
    && make -s -C /src/tools/src install DESTDIR=/out PREFIX=/usr SYSCONFDIR=/etc \
        WITH_WGQUICK=yes WITH_SYSTEMDUNITS=yes WITH_BASHCOMPLETION=no
RUN git clone -q -c advice.detachedHead=false --depth 1 --branch "$AWG_GO_TAG" \
        https://github.com/amnezia-vpn/amneziawg-go /src/go \
    && make -s -C /src/go \
    && install -D -m 0755 /src/go/amneziawg-go /out/usr/bin/amneziawg-go

FROM ubuntu:24.04
ENV container=docker DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install -y --no-install-recommends systemd systemd-sysv dbus iproute2 \
        wireguard-tools kmod iputils-ping procps ca-certificates \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
COPY --from=awg /out/ /
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
