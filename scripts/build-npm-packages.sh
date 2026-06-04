#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
NPM_TEMPLATE_DIR="${ROOT_DIR}/packaging/npm"
NPM_BUILD_DIR="${ROOT_DIR}/build/npm"
LICENSE_FILE="${ROOT_DIR}/LICENSE"

build_target() {
  local goos="$1"
  local goarch="$2"
  local output_name="$3"
  local package_dir="$4"
  local binary_name="$5"

  echo "==> building ${goos}/${goarch}"
  mkdir -p "${DIST_DIR}" "${package_dir}/bin"

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build -o "${DIST_DIR}/${output_name}" "${ROOT_DIR}/."

  cp "${DIST_DIR}/${output_name}" "${package_dir}/bin/${binary_name}"
  cp "${LICENSE_FILE}" "${package_dir}/LICENSE"

  if [[ "${goos}" != "windows" ]]; then
    chmod +x "${package_dir}/bin/${binary_name}"
  fi
}

cd "${ROOT_DIR}"

command -v go >/dev/null 2>&1 || {
  echo "go is required"
  exit 1
}

rm -rf "${NPM_BUILD_DIR}"
mkdir -p "${NPM_BUILD_DIR}"
cp -R "${NPM_TEMPLATE_DIR}/." "${NPM_BUILD_DIR}/"

build_target "darwin"  "arm64" "localtun-darwin-arm64"     "${NPM_BUILD_DIR}/localtun-darwin-arm64" "localtun"
build_target "darwin"  "amd64" "localtun-darwin-x64"       "${NPM_BUILD_DIR}/localtun-darwin-x64"   "localtun"
build_target "linux"   "amd64" "localtun-linux-x64"        "${NPM_BUILD_DIR}/localtun-linux-x64"    "localtun"
build_target "linux"   "arm64" "localtun-linux-arm64"      "${NPM_BUILD_DIR}/localtun-linux-arm64"  "localtun"
build_target "windows" "amd64" "localtun-win32-x64.exe"    "${NPM_BUILD_DIR}/localtun-win32-x64"    "localtun.exe"

cp "${LICENSE_FILE}" "${NPM_BUILD_DIR}/localtun/LICENSE"

echo "==> npm package artifacts are ready in ${NPM_BUILD_DIR}"
