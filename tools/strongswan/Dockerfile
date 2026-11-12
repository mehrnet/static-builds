# Portable strongSwan charon + swanctl for sheed-node.
#
# Monolithic build: all enabled plugins are linked into libstrongswan/libcharon,
# so there is no runtime plugin directory to locate. The tree ships the musl
# loader and every runtime library under bin/ and lib/; wrappers invoke the
# loader with --library-path.
ARG STRONGSWAN_VERSION
FROM alpine:3.20 AS build
ARG STRONGSWAN_VERSION

RUN apk add --no-cache \
    build-base \
    autoconf automake libtool pkgconf gperf bison flex \
    linux-headers \
    openssl-dev \
    curl tar \
    binutils

WORKDIR /src
RUN curl -fsSL "https://github.com/strongswan/strongswan/archive/refs/tags/${STRONGSWAN_VERSION}.tar.gz" -o strongswan.tar.gz \
 && tar xzf strongswan.tar.gz --strip-components=1 \
 && rm strongswan.tar.gz

RUN ./autogen.sh \
 && ./configure \
      --prefix=/opt/sheed-node \
      --sysconfdir=/opt/sheed-node/etc \
      --disable-defaults \
      --disable-nls \
      --enable-charon \
      --enable-swanctl \
      --enable-vici \
      --enable-monolithic \
      --enable-ikev1 \
      --enable-ikev2 \
      --enable-openssl \
      --enable-kernel-netlink \
      --enable-socket-default \
      --enable-pem \
      --enable-pubkey \
      --enable-x509 \
      --enable-constraints \
      --enable-eap-mschapv2 \
      --enable-eap-identity \
      --enable-md4 \
      --enable-sha1 \
      --enable-sha2 \
      --enable-aes \
      --enable-des \
      --enable-hmac \
      --enable-random \
      --enable-nonce \
      --enable-pkcs1 \
      --enable-pkcs8 \
 && make -j"$(nproc)" \
 && DESTDIR=/out-root make install

RUN <<'EOF'
set -eu
mkdir -p /out/bin /out/lib/ipsec
cp /out-root/opt/sheed-node/libexec/ipsec/charon /out/bin/charon.bin
cp /out-root/opt/sheed-node/sbin/swanctl /out/bin/swanctl.bin
cp -a /out-root/opt/sheed-node/lib/ipsec/*.so* /out/lib/ipsec/
arch=$(uname -m)
case "$arch" in
  x86_64) musl=ld-musl-x86_64.so.1 ;;
  aarch64) musl=ld-musl-aarch64.so.1 ;;
  *) echo "unsupported arch $arch" >&2; exit 1 ;;
esac
cp -L "/lib/$musl" /out/lib/
todo="/out/bin/charon.bin /out/bin/swanctl.bin /out/lib/ipsec/*.so*"
while [ -n "$todo" ]; do
  next=""
  for file in $todo; do
    for soname in $(readelf -d "$file" 2>/dev/null | sed -n 's/.*NEEDED.*\[\(.*\)\].*/\1/p'); do
      case "$soname" in
        libc.musl-*|ld-musl-*) continue ;;
      esac
      [ -e "/out/lib/$soname" ] && continue
      [ -e "/out/lib/ipsec/$soname" ] && continue
      src=$(find /lib /usr/lib -name "$soname" -print -quit 2>/dev/null)
      if [ -n "$src" ]; then
        cp -L "$src" "/out/lib/$soname"
        next="$next /out/lib/$soname"
      fi
    done
  done
  todo="$next"
done
for name in charon swanctl; do
  cat > "/out/bin/$name" <<WRAPPER
#!/bin/sh
set -eu
here=\$(CDPATH= cd -- "\$(dirname -- "\$0")" && pwd)
exec "\$here/../lib/$musl" --library-path "\$here/../lib:\$here/../lib/ipsec" "\$here/$name.bin" "\$@"
WRAPPER
  chmod +x "/out/bin/$name"
done
strip /out/bin/*.bin /out/lib/ipsec/*.so* /out/lib/*.so* || true
EOF

FROM scratch AS export
COPY --from=build /out/ /
