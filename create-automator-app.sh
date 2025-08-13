#!/bin/bash

# Automator 앱 생성 스크립트

echo "🤖 Automator 앱 생성 중..."

# AppleScript 생성
cat > build/launch-gui.applescript << 'EOF'
on run
    set appPath to (path to me as text)
    set appPOSIX to POSIX path of appPath
    set appDir to do shell script "dirname " & quoted form of appPOSIX
    
    -- 실행 파일 경로 설정
    set execPath to appDir & "/bmw-monitor-gui"
    
    -- 파일 존재 확인
    try
        do shell script "test -f " & quoted form of execPath
        -- 파일이 있으면 실행
        do shell script quoted form of execPath & " > /dev/null 2>&1 &"
    on error
        -- 파일이 없으면 에러 메시지
        display dialog "실행 파일을 찾을 수 없습니다!" & return & return & "경로: " & execPath buttons {"확인"} default button 1 with icon stop
    end try
end run
EOF

# AppleScript를 앱으로 컴파일
osacompile -o "build/BMW Monitor Launcher.app" build/launch-gui.applescript

# 실행 파일을 앱 번들 안에 복사
cp build/bmw-monitor-gui "build/BMW Monitor Launcher.app/Contents/MacOS/"
cp -r configs "build/BMW Monitor Launcher.app/Contents/Resources/"

echo "✅ Automator 앱 생성 완료!"

# 정리
rm build/launch-gui.applescript

echo ""
echo "📌 생성된 앱들:"
echo "1. BMW Monitor.app - 기본 앱 번들"
echo "2. BMW Monitor Launcher.app - Automator 앱"
echo ""
echo "두 앱 모두 더블클릭으로 실행 가능합니다!"