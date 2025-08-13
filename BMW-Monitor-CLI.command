#!/bin/bash

# BMW 드라이빙 센터 모니터 CLI 실행 스크립트
# 이 파일을 더블클릭하면 터미널이 열리면서 프로그램이 실행됩니다

# 스크립트가 있는 디렉토리로 이동
cd "$(dirname "$0")"

# 실행 파일 확인
if [ -f "build/bmw-monitor-cli" ]; then
    echo "========================================="
    echo "   BMW 드라이빙 센터 모니터 CLI"
    echo "========================================="
    echo ""
    echo "🚗 프로그램을 시작합니다..."
    echo "종료하려면 Ctrl+C를 누르세요."
    echo ""
    echo "========================================="
    echo ""
    
    # CLI 실행
    ./build/bmw-monitor-cli
    
elif [ -f "bmw-monitor-cli" ]; then
    # build 디렉토리가 없는 경우
    echo "========================================="
    echo "   BMW 드라이빙 센터 모니터 CLI"
    echo "========================================="
    echo ""
    echo "🚗 프로그램을 시작합니다..."
    echo "종료하려면 Ctrl+C를 누르세요."
    echo ""
    echo "========================================="
    echo ""
    
    ./bmw-monitor-cli
    
else
    echo "❌ 실행 파일을 찾을 수 없습니다!"
    echo "먼저 './build.sh'를 실행하여 빌드하세요."
    echo ""
    echo "Enter 키를 눌러 종료..."
    read
fi