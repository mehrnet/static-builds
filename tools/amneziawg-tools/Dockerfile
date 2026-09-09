# Static AmneziaWG userspace tools (awg + awg-quick) against musl (Alpine),
# same approach as tools/openvpn -- there's no official static binary
# distribution, so we build awg(8) from source, statically, ourselves.
#
# awg-quick is a bash script (src/wg-quick/linux.bash), not compiled; it's
# copied straight out and shipped alongside. At runtime it tries the
# `amneziawg` kernel module first and, when that's absent, auto-falls back
# to running `amneziawg-go` (the userspace daemon from the sibling tool) --
# no env var or config needed, see the fork's linux.bash.
ARG AWG_TOOLS_VERSION
FROM alpine:3.20 AS build
ARG AWG_TOOLS_VERSION

RUN apk add --no-cache build-base linux-headers curl tar

WORKDIR /src
RUN curl -fsSL "https://github.com/amnezia-vpn/amneziawg-tools/archive/refs/tags/v${AWG_TOOLS_VERSION}.tar.gz" -o awg-tools.tar.gz \
 && tar xzf awg-tools.tar.gz --strip-components=1 \
 && rm awg-tools.tar.gz

# amneziawg-tools vendors its own AmneziaWG-aware linux/wireguard.h under
# src/uapi/linux/ (it carries the awg junk-packet/header attributes the
# stock kernel header lacks). The Makefile adds `-I uapi/linux` so the
# vendored copy wins -- but a system linux/wireguard.h from linux-headers
# can still get pulled in transitively and clash (conflicting enum
# redefinition). Dropping the system copy guarantees the vendored,
# awg-carrying header is the only one in play.
RUN rm -f /usr/include/linux/wireguard.h

RUN make -C src -j"$(nproc)" LDFLAGS="-static" \
 && strip src/awg

FROM scratch AS export
COPY --from=build /src/src/awg /awg
COPY --from=build /src/src/wg-quick/linux.bash /awg-quick
