@echo off
setlocal enabledelayedexpansion

REM BMW 드라이빙 센터 모니터 빌드 스크립트 (Windows)

echo =========================================
echo    BMW 드라이빙 센터 모니터 빌드
echo =========================================
echo.

REM 빌드 디렉토리 생성
set BUILD_DIR=build
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

REM Go 버전 확인
echo 📦 Go 버전 확인...
go version
echo.

REM 의존성 다운로드
echo 📥 의존성 다운로드...
go mod download
echo.

REM CLI 빌드
echo 🔨 CLI 버전 빌드 중...
go build -ldflags="-s -w" -o %BUILD_DIR%\bmw-monitor-cli.exe cmd\cli\main.go

if %ERRORLEVEL% EQU 0 (
    echo ✅ CLI 빌드 성공: %BUILD_DIR%\bmw-monitor-cli.exe
) else (
    echo ❌ CLI 빌드 실패
    exit /b 1
)

REM GUI 빌드
echo.
echo 🔨 GUI 버전 빌드 중...
go build -ldflags="-s -w -H=windowsgui" -o %BUILD_DIR%\bmw-monitor-gui.exe cmd\gui\*.go

if %ERRORLEVEL% EQU 0 (
    echo ✅ GUI 빌드 성공: %BUILD_DIR%\bmw-monitor-gui.exe
) else (
    echo ❌ GUI 빌드 실패
    exit /b 1
)

REM 설정 파일 복사
echo.
echo 📄 설정 파일 복사 중...
xcopy /E /I /Y configs %BUILD_DIR%\configs
echo ✅ 설정 파일 복사 완료

REM README 생성
echo.
echo 📝 빌드 정보 생성 중...

(
echo # BMW 드라이빙 센터 예약 모니터
echo.
echo 빌드 날짜: %date% %time%
echo 시스템: Windows
echo.
echo ## 실행 방법
echo.
echo ### GUI 버전
echo ```
echo bmw-monitor-gui.exe
echo ```
echo.
echo ### CLI 버전
echo ```
echo # 기본 실행
echo bmw-monitor-cli.exe
echo.
echo # 설정 파일 지정
echo bmw-monitor-cli.exe -config configs\config.yaml
echo.
echo # 백그라운드 모드 비활성화 ^(브라우저 창 표시^)
echo bmw-monitor-cli.exe -headless=false
echo.
echo # 확인 간격 변경 ^(초 단위^)
echo bmw-monitor-cli.exe -interval 300
echo.
echo # 사용 가능한 프로그램 목록 보기
echo bmw-monitor-cli.exe -list-programs
echo ```
echo.
echo ## 설정 파일
echo.
echo configs\config.yaml 파일을 수정하여 다음을 설정하세요:
echo - 로그인 정보 ^(username, password^)
echo - 이메일 설정 ^(SMTP^)
echo - 모니터링할 프로그램 목록
echo.
echo ## 주의사항
echo.
echo 1. 첫 실행 전 Playwright 브라우저를 설치해야 합니다:
echo    ```
echo    go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
echo    ```
echo.
echo 2. 세션은 %%USERPROFILE%%\.bmw-driving-center\browser-state\에 저장됩니다.
echo.
echo 3. 반복적인 로그인은 캡챠를 유발할 수 있으므로 세션을 유지합니다.
) > %BUILD_DIR%\README.md

echo ✅ 빌드 정보 생성 완료

REM Playwright 브라우저 확인
echo.
echo 🌐 Playwright 브라우저 확인...
if not exist "%USERPROFILE%\AppData\Local\ms-playwright" (
    echo ⚠️  Playwright 브라우저가 설치되지 않았습니다.
    echo    다음 명령으로 설치하세요:
    echo    go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
) else (
    echo ✅ Playwright 브라우저 설치됨
)

REM 최종 결과
echo.
echo =========================================
echo 🎉 빌드 완료!
echo =========================================
echo.
echo 빌드된 파일:
dir /B %BUILD_DIR%\bmw-monitor-*.exe
echo.
echo 실행 방법:
echo   GUI: %BUILD_DIR%\bmw-monitor-gui.exe
echo   CLI: %BUILD_DIR%\bmw-monitor-cli.exe
echo.
echo 설정 파일 위치: %BUILD_DIR%\configs\config.yaml
echo =========================================

pause