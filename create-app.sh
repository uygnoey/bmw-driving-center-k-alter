#!/bin/bash

# macOS 앱 번들 생성 스크립트

APP_NAME="BMW Monitor"
APP_DIR="build/${APP_NAME}.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

echo "🍎 macOS 앱 번들 생성 중..."

# 기존 앱 번들 삭제
rm -rf "${APP_DIR}"

# 디렉토리 구조 생성
mkdir -p "${MACOS_DIR}"
mkdir -p "${RESOURCES_DIR}"

# 실행 파일 복사
cp build/bmw-monitor-gui "${MACOS_DIR}/bmw-monitor"

# 설정 파일 디렉토리 복사
cp -r configs "${RESOURCES_DIR}/"

# Info.plist 생성
cat > "${CONTENTS_DIR}/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>BMW Monitor</string>
    <key>CFBundleDisplayName</key>
    <string>BMW 드라이빙 센터 모니터</string>
    <key>CFBundleIdentifier</key>
    <string>com.bmw.driving-center-monitor</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>????</string>
    <key>CFBundleExecutable</key>
    <string>bmw-monitor</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.12</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.utilities</string>
</dict>
</plist>
EOF

# 간단한 아이콘 생성 (실제 아이콘이 없으므로 더미 파일)
touch "${RESOURCES_DIR}/AppIcon.icns"

# 실행 권한 설정
chmod +x "${MACOS_DIR}/bmw-monitor"

echo "✅ 앱 번들 생성 완료: ${APP_DIR}"
echo ""
echo "📌 사용 방법:"
echo "1. Finder에서 'build' 폴더 열기"
echo "2. '${APP_NAME}.app' 더블클릭"
echo ""
echo "⚠️  첫 실행 시 보안 경고가 나타날 수 있습니다:"
echo "   시스템 환경설정 → 보안 및 개인정보 보호 → '확인 없이 열기' 클릭"

# Finder에서 build 폴더 열기
open build/