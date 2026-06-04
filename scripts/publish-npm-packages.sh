#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NPM_BUILD_DIR="${ROOT_DIR}/build/npm"

publish_package() {
  local package_dir="$1"
  local package_name
  local version

  package_name="$(node -p "require('${package_dir}/package.json').name")"
  version="$(node -p "require('${package_dir}/package.json').version")"

  if npm view "${package_name}@${version}" version >/dev/null 2>&1; then
    echo "==> ${package_name}@${version} already exists, skipping"
    return
  fi

  echo "==> publishing ${package_dir}"
  (
    cd "${package_dir}"
    npm publish --access public
  )
}

if [[ ! -d "${NPM_BUILD_DIR}" ]]; then
  echo "Missing ${NPM_BUILD_DIR}. Run ./scripts/build-npm-packages.sh first."
  exit 1
fi

publish_package "${NPM_BUILD_DIR}/localtun-darwin-arm64"
publish_package "${NPM_BUILD_DIR}/localtun-darwin-x64"
publish_package "${NPM_BUILD_DIR}/localtun-linux-x64"
publish_package "${NPM_BUILD_DIR}/localtun-linux-arm64"
publish_package "${NPM_BUILD_DIR}/localtun-win32-x64"
publish_package "${NPM_BUILD_DIR}/localtun"
