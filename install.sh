#!/usr/bin/env sh
set -e

REPO="raihankhan/kubes"
BINARY="kubes"
INSTALL_DIR="/usr/local/bin"

# ── Detect OS & Arch ────────────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
  Linux)   OS_NAME="Linux"  ; EXT="tar.gz" ;;
  Darwin)  OS_NAME="Darwin" ; EXT="tar.gz" ;;
  MINGW*|MSYS*|CYGWIN*) OS_NAME="Windows" ; EXT="zip" ;;
  *)
    echo "Unsupported OS: ${OS}"
    exit 1
    ;;
esac

case "${ARCH}" in
  x86_64|amd64)  ARCH_NAME="x86_64" ;;
  arm64|aarch64) ARCH_NAME="arm64"  ;;
  *)
    echo "Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

# ── Resolve latest version ───────────────────────────────────────────────────
if [ -z "${VERSION}" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' \
    | sed -E 's/.*"([^"]+)".*/\1/')"
fi

if [ -z "${VERSION}" ]; then
  echo "Could not determine the latest version. Set VERSION env var to override."
  exit 1
fi

echo "Installing ${BINARY} ${VERSION} (${OS_NAME}/${ARCH_NAME})…"

# ── Download & install ───────────────────────────────────────────────────────
TARBALL="${BINARY}_${OS_NAME}_${ARCH_NAME}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
TMP="$(mktemp -d)"

cleanup() { rm -rf "${TMP}"; }
trap cleanup EXIT

curl -fsSL "${URL}" -o "${TMP}/${TARBALL}"

if [ "${EXT}" = "tar.gz" ]; then
  tar -xzf "${TMP}/${TARBALL}" -C "${TMP}"
else
  unzip -q "${TMP}/${TARBALL}" -d "${TMP}"
fi

# ── Move binary to install dir ───────────────────────────────────────────────
if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  # Requires sudo
  sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo ""
echo "  kubes ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
echo "  "
echo "  CRITICAL STEP: Add the following to your ~/.zshrc or ~/.bashrc to enable KUBECONFIG switching:"
echo "  kubes() {"
echo "      export KUBES_ENV_FILE=\$(mktemp)"
echo "      command kubes \"\$@\""
echo "      local exit_code=\$?"
echo "      if [ -f \"\$KUBES_ENV_FILE\" ]; then"
echo "          source \"\$KUBES_ENV_FILE\""
echo "          rm -f \"\$KUBES_ENV_FILE\""
echo "      fi"
echo "      return \$exit_code"
echo "  }"
echo "  "
echo "  Restart your terminal and run 'kubes' to get started."
