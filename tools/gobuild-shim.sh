#!/usr/bin/env bash
# gobuild-shim.sh — GoReleaser `gobinary` override.
#
# GoReleaser OSS always compiles binaries itself and cannot consume binaries
# built elsewhere (its `prebuilt` builder is Pro-only). ogcode depends on
# github.com/gen2brain/go-fitz, which only links statically (no runtime
# libmupdf.* needed) when built with CGO_ENABLED=1 against the *target*
# platform's C toolchain. Reliable CGO cross-compilation of that C codebase
# from a single runner is impractical, so the release workflow builds each
# target natively on its own runner and stages the binaries under
# $PREBUILT_DIR/ogcode_<goos>_<goarch>/ogcode[.exe].
#
# This shim makes GoReleaser "build" by copying the matching prebuilt binary
# into the output path it requests, while passing every other go invocation
# (version, env, list, mod, …) straight through to the real `go`. That keeps
# GoReleaser's archive/checksum/brew/winget steps working unchanged.
set -euo pipefail

if [ "${1:-}" = "build" ]; then
    out=""
    prev=""
    for arg in "$@"; do
        if [ "$prev" = "-o" ]; then
            out="$arg"
        fi
        prev="$arg"
    done

    if [ -z "${out}" ]; then
        echo "gobuild-shim: no -o output path in: $*" >&2
        exit 1
    fi
    if [ -z "${PREBUILT_DIR:-}" ]; then
        echo "gobuild-shim: PREBUILT_DIR is not set" >&2
        exit 1
    fi

    src="${PREBUILT_DIR}/ogcode_${GOOS}_${GOARCH}/ogcode"
    if [ "${GOOS}" = "windows" ]; then
        src="${src}.exe"
    fi
    if [ ! -f "${src}" ]; then
        echo "gobuild-shim: prebuilt binary not found: ${src}" >&2
        exit 1
    fi

    mkdir -p "$(dirname "${out}")"
    cp "${src}" "${out}"
    chmod +x "${out}"
    echo "gobuild-shim: ${src} -> ${out} (GOOS=${GOOS} GOARCH=${GOARCH})" >&2
    exit 0
fi

exec go "$@"
