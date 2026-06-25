#!/usr/bin/env sh
set -eu

repo="${TREECTL_REPO:-Knovigator/treectl}"
install_dir="${TREECTL_INSTALL_DIR:-$HOME/.local/bin}"
version="${TREECTL_VERSION:-latest}"

need() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "treectl installer requires $1" >&2
        exit 1
    fi
}

need curl
need tar

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
    darwin|linux) ;;
    *)
        echo "unsupported operating system: $os" >&2
        exit 1
        ;;
esac

arch="$(uname -m)"
case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
        echo "unsupported architecture: $arch" >&2
        exit 1
        ;;
esac

if [ "$version" = "latest" ]; then
    tag="$(
        curl -fsSL "https://api.github.com/repos/${repo}/releases?per_page=100" |
            sed -n 's/.*"tag_name": "\(v[^"]*\)".*/\1/p' |
            head -n 1
    )"
    if [ -z "$tag" ]; then
        echo "could not find a treectl release for ${repo}" >&2
        exit 1
    fi
else
    case "$version" in
        v*) tag="$version" ;;
        *) tag="v$version" ;;
    esac
fi

asset="treectl_${tag}_${os}_${arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${tag}"

tmpdir="$(mktemp -d)"
cleanup() {
    rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

echo "Downloading ${asset} from ${repo} ${tag}"
curl -fL "${base_url}/${asset}" -o "${tmpdir}/${asset}"
curl -fsSL "${base_url}/checksums.txt" -o "${tmpdir}/checksums.txt"

expected="$(grep " ${asset}$" "${tmpdir}/checksums.txt" | awk '{print $1}')"
if [ -z "$expected" ]; then
    echo "could not find checksum for ${asset}" >&2
    exit 1
fi

if command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "${tmpdir}/${asset}" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${tmpdir}/${asset}" | awk '{print $1}')"
else
    echo "neither shasum nor sha256sum is available for checksum verification" >&2
    exit 1
fi

if [ "$expected" != "$actual" ]; then
    echo "checksum mismatch for ${asset}" >&2
    exit 1
fi

tar -C "$tmpdir" -xzf "${tmpdir}/${asset}"
mkdir -p "$install_dir"

if command -v install >/dev/null 2>&1; then
    install -m 0755 "${tmpdir}/treectl" "${install_dir}/treectl"
else
    cp "${tmpdir}/treectl" "${install_dir}/treectl"
    chmod 0755 "${install_dir}/treectl"
fi

echo "Installed treectl to ${install_dir}/treectl"
case ":$PATH:" in
    *":${install_dir}:"*) ;;
    *)
        echo "Add ${install_dir} to PATH to run treectl from any directory."
        ;;
esac
