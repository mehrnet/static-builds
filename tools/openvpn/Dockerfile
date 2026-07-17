# Static OpenVPN build against musl (Alpine) -- there's no official static
# binary distribution for OpenVPN the way Xray-core ships one, so this
# actually builds it from source, statically, ourselves. Built from the raw
# GitHub tag archive (not the swupdate.openvpn.org release tarball), so the
# generated build files (./configure etc.) don't exist yet -- autoreconf
# regenerates them.
ARG OPENVPN_VERSION
FROM alpine:3.20 AS build
ARG OPENVPN_VERSION

RUN apk add --no-cache \
    build-base \
    autoconf automake libtool pkgconfig \
    linux-headers \
    openssl-dev \
    lz4-dev \
    lzo-dev \
    curl tar

WORKDIR /src
RUN curl -fsSL "https://github.com/OpenVPN/openvpn/archive/refs/tags/v${OPENVPN_VERSION}.tar.gz" -o openvpn.tar.gz \
 && tar xzf openvpn.tar.gz --strip-components=1 \
 && rm openvpn.tar.gz

# --disable-management/--disable-plugins: this binary only ever needs to
# bring a tunnel up for a probe/test cycle and tear it down -- the
# management interface and plugin loading (auth-pam, down-root, ...) are
# attack surface radar-node's use case has no reason to carry.
RUN autoreconf -fiv \
 && ./configure \
      --enable-static \
      --disable-shared \
      --disable-management \
      --disable-plugins \
      --disable-debug \
      LDFLAGS="-static" \
 && make -j"$(nproc)" \
 && strip src/openvpn/openvpn

FROM scratch AS export
COPY --from=build /src/src/openvpn/openvpn /openvpn
