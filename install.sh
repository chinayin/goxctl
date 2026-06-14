#!/bin/sh
# 安装 goxctl 核心，零 Go 依赖（下载预编译二进制）。
# 用法：curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl/main/install.sh | sh [-s -- <version>]
set -eu

CORE_REPO="chinayin/goxctl"
INSTALL_DIR="${GOXCTL_BIN_DIR:-$HOME/.goxctl/bin}"
VERSION="${1:-latest}"

# 平台探测（仅支持 macOS / Linux，amd64 / arm64）
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) echo "不支持的架构: $arch" >&2; exit 1 ;;
esac
case "$os" in
	darwin | linux) ;;
	*) echo "不支持的系统: $os（install.sh 仅支持 macOS/Linux）" >&2; exit 1 ;;
esac

fetch() { # url dest
	if command -v curl >/dev/null 2>&1; then
		curl -sSfL "$1" -o "$2"
	else
		wget -qO "$2" "$1"
	fi
}

sha256_of() { # file -> sha
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

asset_url() { # asset -> url
	if [ "$VERSION" = latest ]; then
		echo "https://github.com/$CORE_REPO/releases/latest/download/$1"
	else
		echo "https://github.com/$CORE_REPO/releases/download/$VERSION/$1"
	fi
}

asset="goxctl_${os}_${arch}.tar.gz"
tmp=$(mktemp -d)
echo "下载 goxctl ($VERSION, $os/$arch) ..."
fetch "$(asset_url "$asset")" "$tmp/$asset"
fetch "$(asset_url checksums.txt)" "$tmp/checksums.txt"

want=$(grep " $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
got=$(sha256_of "$tmp/$asset")
if [ -z "$want" ] || [ "$want" != "$got" ]; then
	echo "校验失败: $asset" >&2
	exit 1
fi

mkdir -p "$INSTALL_DIR"
tar -xzf "$tmp/$asset" -C "$INSTALL_DIR" goxctl
chmod +x "$INSTALL_DIR/goxctl"
rm -rf "$tmp"

echo
echo "完成。若 goxctl 不在 PATH，请加入："
echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
echo "试试：goxctl version"
