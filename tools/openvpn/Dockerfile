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
    openssl-libs-static \
    lz4-dev \
    lz4-static \
    lzo-dev \
    libcap-ng-dev libcap-ng-static \
    curl tar

WORKDIR /src
RUN curl -fsSL "https://github.com/OpenVPN/openvpn/archive/refs/tags/v${OPENVPN_VERSION}.tar.gz" -o openvpn.tar.gz \
 && tar xzf openvpn.tar.gz --strip-components=1 \
 && rm openvpn.tar.gz

# --disable-management/--disable-plugins: this binary only ever needs to
# bring a tunnel up for a probe/test cycle and tear it down -- the
# management interface and plugin loading (auth-pam, down-root, ...) are
# attack surface radar-node's use case has no reason to carry.
# --disable-dco: Data Channel Offload needs libnl-genl-3.0 (kernel-module
# acceleration) that has nothing to do with this binary's job -- a short
# probe/test connection, not a high-throughput production tunnel.
#
# LDFLAGS="-static" at configure time is what the small AC_CHECK_LIB test
# programs need to compile/link directly against a real linker flag; it's
# NOT enough on its own to make the *final* binary static, though, since
# that link goes through libtool -- libtool intercepts a plain "-static"
# for its own (different) meaning and silently keeps linking shared libs
# otherwise. Overriding LDFLAGS again at the `make` invocation itself,
# to libtool's own "-all-static", is what actually forces the real
# openvpn binary's final link to be fully static.
RUN autoreconf -fiv \
 && ./configure \
      --enable-static \
      --disable-shared \
      --disable-management \
      --disable-plugins \
      --disable-debug \
      --disable-dco \
      LDFLAGS="-static" \
 && make -j"$(nproc)" LDFLAGS="-all-static" \
 && strip src/openvpn/openvpn

FROM scratch AS export
COPY --from=build /src/src/openvpn/openvpn /openvpn
