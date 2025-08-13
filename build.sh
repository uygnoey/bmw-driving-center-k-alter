#!/bin/bash

# BMW 드라이빙 센터 모니터 빌드 스크립트

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 빌드 디렉토리 생성
BUILD_DIR="build"
mkdir -p ${BUILD_DIR}

echo "========================================="
echo "   BMW 드라이빙 센터 모니터 빌드"
echo "========================================="
echo ""

# 현재 OS 확인
OS=$(uname -s)
ARCH=$(uname -m)

echo "🖥️  시스템: ${OS} ${ARCH}"
echo ""

# Go 버전 확인
echo "📦 Go 버전 확인..."
go version
echo ""

# 플랫폼별 설정
case "$OS" in
    Darwin*)
        echo "🍎 macOS 빌드 설정"
        GUI_FLAGS='-ldflags="-s -w"'
        if [ "$ARCH" = "arm64" ]; then
            export GOARCH=arm64
            echo "   Apple Silicon (M1/M2) 감지됨"
        else
            export GOARCH=amd64
            echo "   Intel Mac 감지됨"
        fi
        ;;
    Linux*)
        echo "🐧 Linux 빌드 설정"
        GUI_FLAGS='-ldflags="-s -w"'
        export CGO_ENABLED=1
        ;;
    MINGW*|MSYS*|CYGWIN*)
        echo "🪟 Windows 빌드 설정"
        GUI_FLAGS='-ldflags="-s -w -H=windowsgui"'
        ;;
    *)
        echo -e "${RED}❌ 지원하지 않는 운영체제: $OS${NC}"
        exit 1
        ;;
esac

# 의존성 다운로드
echo "📥 의존성 다운로드..."
go mod download
echo ""

# CLI 빌드
echo "🔨 CLI 버전 빌드 중..."
go build -ldflags="-s -w" -o ${BUILD_DIR}/bmw-monitor-cli cmd/cli/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ CLI 빌드 성공: ${BUILD_DIR}/bmw-monitor-cli${NC}"
    # 실행 권한 부여
    chmod +x ${BUILD_DIR}/bmw-monitor-cli
else
    echo -e "${RED}❌ CLI 빌드 실패${NC}"
    exit 1
fi

# GUI 빌드
echo ""
echo "🔨 GUI 버전 빌드 중..."
go build -ldflags="-s -w" -o ${BUILD_DIR}/bmw-monitor-gui cmd/gui/*.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ GUI 빌드 성공: ${BUILD_DIR}/bmw-monitor-gui${NC}"
    # 실행 권한 부여
    chmod +x ${BUILD_DIR}/bmw-monitor-gui
else
    echo -e "${RED}❌ GUI 빌드 실패${NC}"
    exit 1
fi

# 설정 파일 복사
echo ""
echo "📄 설정 파일 복사 중..."
cp -r configs ${BUILD_DIR}/
echo -e "${GREEN}✅ 설정 파일 복사 완료${NC}"

# Playwright 브라우저 확인
echo ""
echo "🌐 Playwright 브라우저 확인..."
if [ ! -d "$HOME/Library/Caches/ms-playwright" ] && [ ! -d "$HOME/.cache/ms-playwright" ]; then
    echo -e "${YELLOW}⚠️  Playwright 브라우저가 설치되지 않았습니다.${NC}"
    echo "   다음 명령으로 설치하세요:"
    echo "   go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium"
else
    echo -e "${GREEN}✅ Playwright 브라우저 설치됨${NC}"
fi

# 빌드 정보 생성
echo ""
echo "📝 빌드 정보 생성 중..."
cat > ${BUILD_DIR}/README.md << EOF
# BMW 드라이빙 센터 예약 모니터

빌드 날짜: $(date '+%Y-%m-%d %H:%M:%S')
시스템: ${OS} ${ARCH}

## 실행 방법

### GUI 버전
\`\`\`bash
./bmw-monitor-gui
\`\`\`

### CLI 버전
\`\`\`bash
# 기본 실행
./bmw-monitor-cli

# 설정 파일 지정
./bmw-monitor-cli -config configs/config.yaml

# 백그라운드 모드 비활성화 (브라우저 창 표시)
./bmw-monitor-cli -headless=false

# 확인 간격 변경 (초 단위)
./bmw-monitor-cli -interval 300

# 사용 가능한 프로그램 목록 보기
./bmw-monitor-cli -list-programs
\`\`\`

## 설정 파일

configs/config.yaml 파일을 수정하여 다음을 설정하세요:
- 로그인 정보 (username, password)
- 이메일 설정 (SMTP)
- 모니터링할 프로그램 목록

## 주의사항

1. 첫 실행 전 Playwright 브라우저를 설치해야 합니다:
   \`\`\`bash
   go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
   \`\`\`

2. 세션은 ~/.bmw-driving-center/browser-state/에 저장됩니다.

3. 반복적인 로그인은 캡챠를 유발할 수 있으므로 세션을 유지합니다.
EOF

echo -e "${GREEN}✅ 빌드 정보 생성 완료${NC}"

# macOS 앱 번들 생성
if [ "$OS" = "Darwin" ]; then
    echo ""
    echo "🍎 macOS 앱 번들 생성 중..."
    
    APP_NAME="BMW Monitor"
    APP_DIR="${BUILD_DIR}/${APP_NAME}.app"
    CONTENTS_DIR="${APP_DIR}/Contents"
    MACOS_DIR="${CONTENTS_DIR}/MacOS"
    RESOURCES_DIR="${CONTENTS_DIR}/Resources"
    
    # 기존 앱 번들 삭제 및 디렉토리 생성
    rm -rf "${APP_DIR}"
    mkdir -p "${MACOS_DIR}"
    mkdir -p "${RESOURCES_DIR}"
    
    # 실행 파일 및 설정 복사
    cp ${BUILD_DIR}/bmw-monitor-gui "${MACOS_DIR}/bmw-monitor"
    cp -r configs "${RESOURCES_DIR}/"
    
    # Info.plist 생성
    cat > "${CONTENTS_DIR}/Info.plist" << PLIST
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
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
PLIST
    
    chmod +x "${MACOS_DIR}/bmw-monitor"
    echo -e "${GREEN}✅ macOS 앱 번들 생성 완료${NC}"
fi

# 최종 결과
echo ""
echo "========================================="
echo -e "${GREEN}🎉 빌드 완료!${NC}"
echo "========================================="
echo ""
echo "빌드된 파일:"
ls -lh ${BUILD_DIR}/bmw-monitor-*
if [ "$OS" = "Darwin" ]; then
    echo ""
    echo "macOS 앱:"
    echo "  • ${BUILD_DIR}/BMW Monitor.app (더블클릭 실행 가능)"
fi
echo ""
echo "실행 방법:"
echo "  GUI: ./${BUILD_DIR}/bmw-monitor-gui"
echo "  CLI: ./${BUILD_DIR}/bmw-monitor-cli"
if [ "$OS" = "Darwin" ]; then
    echo "  앱: open '${BUILD_DIR}/BMW Monitor.app'"
fi
echo ""
echo "설정 파일 위치: ${BUILD_DIR}/configs/config.yaml"
echo "========================================="