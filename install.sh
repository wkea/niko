#!/bin/sh
# Niko 一键安装脚本（Linux）
#
#   curl -fsSL https://raw.githubusercontent.com/wkea/niko/main/install.sh | sh
#
# 参数：
#   --dir DIR         安装目录（默认 /usr/local/bin；不可写时用 ~/.local/bin）
#   --version TAG     指定版本，例如 v0.1.0（默认取最新 Release）
#   --skip-onboard    只装程序，不执行首次初始化
#   -h, --help        显示帮助
#
# 环境变量：
#   NIKO_INSTALL_DIR  等同 --dir
#   NIKO_VERSION      等同 --version
#   NIKO_BASE_URL     下载源前缀，默认 https://github.com（可指向镜像站）
#
# 重复执行即升级；配置与记忆保存在 ~/.picoclaw，不会被覆盖。

set -eu

REPO="wkea/niko"
BASE_URL="${NIKO_BASE_URL:-https://github.com}"
VERSION="${NIKO_VERSION:-latest}"
INSTALL_DIR="${NIKO_INSTALL_DIR:-}"
DO_ONBOARD=1

info() { printf '  %s\n' "$*"; }
step() { printf '==> %s\n' "$*"; }
die() { printf '错误: %s\n' "$*" >&2; exit 1; }

usage() {
	cat <<'EOF'
Niko 一键安装脚本

用法:
  install.sh [选项]

选项:
  --dir DIR         安装目录（默认 /usr/local/bin，不可写时用 ~/.local/bin）
  --version TAG     指定版本，例如 v0.1.0（默认取最新 Release）
  --skip-onboard    只装程序，不执行首次初始化
  -h, --help        显示本帮助

环境变量:
  NIKO_INSTALL_DIR  等同 --dir
  NIKO_VERSION      等同 --version
  NIKO_BASE_URL     下载源前缀，默认 https://github.com

示例:
  curl -fsSL https://raw.githubusercontent.com/wkea/niko/main/install.sh | sh
  sh install.sh --dir ~/bin --version v0.1.0
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
		--dir) [ $# -ge 2 ] || die "--dir 需要一个目录"; INSTALL_DIR="$2"; shift 2 ;;
		--dir=*) INSTALL_DIR="${1#*=}"; shift ;;
		--version) [ $# -ge 2 ] || die "--version 需要一个版本号"; VERSION="$2"; shift 2 ;;
		--version=*) VERSION="${1#*=}"; shift ;;
		--skip-onboard) DO_ONBOARD=0; shift ;;
		-h|--help) usage; exit 0 ;;
		*) die "未知参数: $1（用 --help 查看用法）" ;;
	esac
done

# ---------- 环境检查 ----------

OS="$(uname -s 2>/dev/null || echo unknown)"
case "$OS" in
	Linux) ;;
	Darwin)
		die "预编译包只提供 Linux。macOS 请用一行编译：
  git clone https://github.com/${REPO}.git && cd niko
  go build -o niko ./cmd/picoclaw"
		;;
	*)
		die "不支持的系统: $OS（预编译包只提供 Linux）"
		;;
esac

MACHINE="$(uname -m 2>/dev/null || echo unknown)"
case "$MACHINE" in
	x86_64|amd64) ARCH=amd64 ;;
	aarch64|arm64|armv8*|armv8l) ARCH=arm64 ;;
	armv7l|armv7*|armv6l|armhf|arm) ARCH=armv7 ;;
	*) die "不支持的架构: $MACHINE（已提供 amd64 / arm64 / armv7）" ;;
esac

ASSET="niko-linux-${ARCH}.tar.gz"
if [ "$VERSION" = "latest" ]; then
	URL="${BASE_URL}/${REPO}/releases/latest/download/${ASSET}"
else
	URL="${BASE_URL}/${REPO}/releases/download/${VERSION}/${ASSET}"
fi

if command -v curl >/dev/null 2>&1; then
	DOWNLOAD() { curl -fL --retry 3 --connect-timeout 15 -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	DOWNLOAD() { wget -O "$2" "$1"; }
else
	die "需要 curl 或 wget 才能下载"
fi
command -v tar >/dev/null 2>&1 || die "需要 tar 才能解压"

step "检测到 ${OS} / ${MACHINE}，安装 ${ASSET}"
info "下载地址: $URL"

TMPDIR_INSTALL="$(mktemp -d 2>/dev/null || mktemp -d -t niko)"
trap 'rm -rf "$TMPDIR_INSTALL"' EXIT INT TERM

DOWNLOAD "$URL" "$TMPDIR_INSTALL/$ASSET" || die "下载失败。国内网络可换镜像：
  NIKO_BASE_URL=https://<镜像站前缀> sh install.sh"

tar -xzf "$TMPDIR_INSTALL/$ASSET" -C "$TMPDIR_INSTALL" niko 2>/dev/null \
	|| tar -xzf "$TMPDIR_INSTALL/$ASSET" -C "$TMPDIR_INSTALL"
[ -f "$TMPDIR_INSTALL/niko" ] || die "压缩包内未找到 niko"
chmod 755 "$TMPDIR_INSTALL/niko"

# ---------- 选择安装目录 ----------

SUDO=""
if [ -z "$INSTALL_DIR" ]; then
	if [ -w /usr/local/bin ] || [ "$(id -u)" = "0" ]; then
		INSTALL_DIR=/usr/local/bin
	elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
		INSTALL_DIR=/usr/local/bin
		SUDO="sudo"
	else
		INSTALL_DIR="$HOME/.local/bin"
	fi
fi

step "安装到 ${INSTALL_DIR}/niko"
$SUDO mkdir -p "$INSTALL_DIR" || die "无法创建目录 $INSTALL_DIR"
if [ ! -w "$INSTALL_DIR" ] && [ -z "$SUDO" ]; then
	die "目录不可写: $INSTALL_DIR（用 --dir 换一个，或在命令前加 sudo）"
fi

if command -v install >/dev/null 2>&1; then
	$SUDO install -m 755 "$TMPDIR_INSTALL/niko" "$INSTALL_DIR/niko"
else
	$SUDO cp "$TMPDIR_INSTALL/niko" "$INSTALL_DIR/niko"
	$SUDO chmod 755 "$INSTALL_DIR/niko"
fi

DEST="$INSTALL_DIR/niko"
"$DEST" version >/dev/null 2>&1 || die "安装后无法运行 $DEST，请检查架构是否匹配"

# ---------- PATH 提示 ----------

case ":${PATH}:" in
	*":${INSTALL_DIR}:"*) ;;
	*)
		printf '\n提示: %s 不在 PATH 中，请把下面一行加进 ~/.bashrc 或 ~/.zshrc：\n' "$INSTALL_DIR"
		printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
		;;
esac

# ---------- 首次初始化 ----------

CONFIG="${PICOCLAW_HOME:-$HOME/.picoclaw}/config.json"
if [ "$DO_ONBOARD" = "1" ] && [ ! -f "$CONFIG" ]; then
	step "首次运行，初始化配置与工作区"
	"$DEST" onboard || info "初始化未完成，稍后可手动执行: niko onboard"
fi

printf '\n'
step "安装完成"
info "已安装: $DEST"
printf '\n下一步（三选一）：\n'
printf '  1) 接一个模型（示例为 DeepSeek）：\n'
printf '     niko model add -b https://api.deepseek.com -k sk-你的Key -m deepseek-flash -n deepseek-flash\n'
printf '  2) 直接对话：niko\n'
printf '  3) 常驻接聊天软件：niko gateway\n'
printf '\n文档: https://github.com/%s\n' "$REPO"
