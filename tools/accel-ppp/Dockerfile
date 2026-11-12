# Portable accel-ppp build for sheed-node.
#
# accel-ppp is plugin-based (the daemon dlopen()s l2tp/auth/ippool/chap-secrets
# modules), so a single fully static executable is not possible without
# patching its loader. Instead this builds against musl (Alpine) and ships a
# self-contained tree:
#
#   bin/accel-pppd       wrapper that runs the musl loader
#   bin/accel-pppd.bin   real daemon
#   bin/accel-cmd        wrapper
#   bin/accel-cmd.bin    real CLI
#   lib/ld-musl-*.so.1   musl loader/libc
#   lib/*.so*            runtime dependencies
#   lib/accel-ppp/*.so   plugins + libtriton
#   share/accel-ppp/     L2TP/PPP dictionaries
#
# The wrapper sets ACCEL_PPP_MODULE_PATH; the loader patch below makes
# accel-ppp honor that environment variable instead of the compile-time path.
ARG ACCEL_PPP_VERSION
FROM alpine:3.20 AS build
ARG ACCEL_PPP_VERSION

RUN apk add --no-cache \
    build-base \
    cmake \
    linux-headers \
    openssl-dev \
    pcre2-dev \
    libnl3-dev \
    libucontext-dev \
    curl \
    tar \
    binutils

WORKDIR /src
RUN curl -fsSL "https://github.com/accel-ppp/accel-ppp/archive/refs/tags/${ACCEL_PPP_VERSION}.tar.gz" -o accel-ppp.tar.gz \
 && tar xzf accel-ppp.tar.gz --strip-components=1 \
 && rm accel-ppp.tar.gz

# Honor ACCEL_PPP_MODULE_PATH so the portable wrapper can point at the
# bundled plugin directory instead of the compile-time install prefix.
RUN sed -i 's|char \*path = MODULE_PATH;|char *path = getenv("ACCEL_PPP_MODULE_PATH");\n\tif (!path \|\| !*path)\n\t\tpath = MODULE_PATH;|' accel-pppd/triton/loader.c

RUN cmake -S . -B build \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_INSTALL_PREFIX=/opt/sheed-node \
      -DRADIUS=FALSE \
      -DSHAPER=FALSE \
      -DLUA=FALSE \
      -DCMAKE_C_FLAGS="-O2" \
 && cmake --build build -j"$(nproc)" \
 && DESTDIR=/out-root cmake --install build

RUN <<'EOF'
set -eu
mkdir -p /out/bin /out/lib /out/lib/accel-ppp /out/share/accel-ppp
cp -a /out-root/opt/sheed-node/share/accel-ppp/. /out/share/accel-ppp/
cp /out-root/opt/sheed-node/sbin/accel-pppd /out/bin/accel-pppd.bin
cp /out-root/opt/sheed-node/bin/accel-cmd /out/bin/accel-cmd.bin
cp /out-root/opt/sheed-node/lib*/accel-ppp/*.so /out/lib/accel-ppp/
arch=$(uname -m)
case "$arch" in
  x86_64) musl=ld-musl-x86_64.so.1 ;;
  aarch64) musl=ld-musl-aarch64.so.1 ;;
  *) echo "unsupported arch $arch" >&2; exit 1 ;;
esac
cp -L "/lib/$musl" /out/lib/
todo="/out/bin/accel-pppd.bin /out/bin/accel-cmd.bin /out/lib/accel-ppp/*.so"
while [ -n "$todo" ]; do
  next=""
  for file in $todo; do
    for soname in $(readelf -d "$file" 2>/dev/null | sed -n 's/.*NEEDED.*\[\(.*\)\].*/\1/p'); do
      case "$soname" in
        libc.musl-*|ld-musl-*) continue ;;
      esac
      [ -e "/out/lib/$soname" ] && continue
      [ -e "/out/lib/accel-ppp/$soname" ] && continue
      src=$(find /lib /usr/lib -name "$soname" -print -quit 2>/dev/null)
      if [ -n "$src" ]; then
        cp -L "$src" "/out/lib/$soname"
        next="$next /out/lib/$soname"
      fi
    done
  done
  todo="$next"
done
for name in accel-pppd accel-cmd; do
  cat > "/out/bin/$name" <<WRAPPER
#!/bin/sh
set -eu
here=\$(CDPATH= cd -- "\$(dirname -- "\$0")" && pwd)
export ACCEL_PPP_MODULE_PATH="\$here/../lib/accel-ppp"
exec "\$here/../lib/$musl" --library-path "\$here/../lib:\$here/../lib/accel-ppp" "\$here/$name.bin" "\$@"
WRAPPER
  chmod +x "/out/bin/$name"
done
strip /out/bin/*.bin /out/lib/accel-ppp/*.so /out/lib/*.so* || true
EOF

FROM scratch AS export
COPY --from=build /out/ /
