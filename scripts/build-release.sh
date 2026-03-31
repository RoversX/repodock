#!/usr/bin/env bash
set -euo pipefail

APP=repodock
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
DIST_DIR="${ROOT_DIR}/dist"
BUILD_DIR="${DIST_DIR}/build"
PLIST_PATH="${BUILD_DIR}/repodock-info.plist"

VERSION=${VERSION:-dev}
COMMIT=${COMMIT:-$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo "none")}
DATE=${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}

LDFLAGS=(
  "-s"
  "-w"
  "-X" "github.com/roversx/repodock/internal/buildinfo.Version=${VERSION}"
  "-X" "github.com/roversx/repodock/internal/buildinfo.Commit=${COMMIT}"
  "-X" "github.com/roversx/repodock/internal/buildinfo.Date=${DATE}"
)

mkdir -p "${BUILD_DIR}"
rm -rf "${BUILD_DIR:?}"/*
rm -f "${DIST_DIR}/${APP}-macos.zip" "${DIST_DIR}/${APP}-linux-amd64.tar.gz" "${DIST_DIR}/${APP}-linux-arm64.tar.gz" "${DIST_DIR}/checksums.txt"

cat > "${PLIST_PATH}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key>
  <string>com.roversx.repodock</string>
  <key>CFBundleExecutable</key>
  <string>${APP}</string>
  <key>CFBundleName</key>
  <string>RepoDock</string>
  <key>CFBundleDisplayName</key>
  <string>RepoDock</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>NSHumanReadableCopyright</key>
  <string>Copyright © 2026 RoversX / CloseX. Licensed under GPL-3.0</string>
</dict>
</plist>
EOF

DARWIN_LDFLAGS="${LDFLAGS[*]} -linkmode external -extldflags '-Wl,-sectcreate,__TEXT,__info_plist,${PLIST_PATH}'"

echo "Building macOS arm64"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
  go build -trimpath -ldflags "${DARWIN_LDFLAGS}" -o "${BUILD_DIR}/${APP}-darwin-arm64" ./cmd/${APP}

echo "Building macOS amd64"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
  go build -trimpath -ldflags "${DARWIN_LDFLAGS}" -o "${BUILD_DIR}/${APP}-darwin-amd64" ./cmd/${APP}

echo "Creating macOS universal binary"
lipo -create \
  -output "${BUILD_DIR}/${APP}" \
  "${BUILD_DIR}/${APP}-darwin-arm64" \
  "${BUILD_DIR}/${APP}-darwin-amd64"

codesign --force --sign - --identifier com.roversx.repodock "${BUILD_DIR}/${APP}"

cp "${ROOT_DIR}/LICENSE" "${BUILD_DIR}/LICENSE"
cp "${ROOT_DIR}/README.md" "${BUILD_DIR}/README.md"

(
  cd "${BUILD_DIR}"
  zip -q -r "${DIST_DIR}/${APP}-macos.zip" "${APP}" LICENSE README.md
)

for target in linux/amd64 linux/arm64; do
  os=${target%/*}
  arch=${target#*/}
  out_dir="${BUILD_DIR}/${APP}-${os}-${arch}"
  mkdir -p "${out_dir}"
  echo "Building ${os} ${arch}"
  GOOS=${os} GOARCH=${arch} CGO_ENABLED=0 \
    go build -trimpath -ldflags "${LDFLAGS[*]}" -o "${out_dir}/${APP}" ./cmd/${APP}
  cp "${ROOT_DIR}/LICENSE" "${out_dir}/LICENSE"
  cp "${ROOT_DIR}/README.md" "${out_dir}/README.md"
  tar -C "${out_dir}" -czf "${DIST_DIR}/${APP}-${os}-${arch}.tar.gz" "${APP}" LICENSE README.md
done

(
  cd "${DIST_DIR}"
  shasum -a 256 \
    "${APP}-macos.zip" \
    "${APP}-linux-amd64.tar.gz" \
    "${APP}-linux-arm64.tar.gz" > checksums.txt
)

echo "Release artifacts written to ${DIST_DIR}"
