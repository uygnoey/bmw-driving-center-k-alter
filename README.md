# BMW 드라이빙 센터 예약 모니터 🚗

BMW 드라이빙 센터(https://driving-center.bmw.co.kr) 예약 상태를 24시간 모니터링하고 예약이 오픈되면 이메일로 알려주는 프로그램입니다.

## 주요 기능 ✨

- 🔐 **OAuth2 로그인**: BMW 고객 계정 2단계 인증 지원
- 💾 **세션 유지**: 한 번 로그인 후 세션 자동 저장/복원
- 📧 **이메일 알림**: 예약 가능 시 즉시 이메일 발송
- 🖥️ **GUI/CLI 지원**: 편리한 GUI와 서버용 CLI 모두 제공
- 🤖 **백그라운드 실행**: 브라우저 창 없이 조용히 실행 가능
- 📋 **프로그램 선택**: 원하는 프로그램만 선택하여 모니터링

## 빠른 시작 🚀

### 1. 빌드된 파일 실행

빌드 디렉토리에서 실행 파일을 찾을 수 있습니다:
- GUI: `build/bmw-monitor-gui`
- CLI: `build/bmw-monitor-cli`

### 2. Playwright 브라우저 설치

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
```

### 3. 설정 파일 수정

`configs/config.yaml` 파일을 열어 다음을 설정하세요:

```yaml
auth:
    username: your-bmw-id@email.com  # BMW ID
    password: your-password           # 비밀번호

email:
    smtp:
        host: smtp.gmail.com
        port: 587
        username: your-email@gmail.com
        password: your-app-password  # Gmail 앱 비밀번호
    from: your-email@gmail.com
    to:
        - recipient@example.com

programs:  # 모니터링할 프로그램 선택
    - name: M Core
      keywords:
        - M Core
        - M 코어
```

### 4. 실행

#### GUI 버전
```bash
./build/bmw-monitor-gui
```

#### CLI 버전
```bash
# 기본 실행
./build/bmw-monitor-cli

# 브라우저 창 표시
./build/bmw-monitor-cli -headless=false

# 확인 간격 변경 (초)
./build/bmw-monitor-cli -interval 300

# 사용 가능한 프로그램 목록 보기
./build/bmw-monitor-cli -list-programs
```

## 직접 빌드하기 🔨

### 필요 사항
- Go 1.20 이상
- Make (선택사항)

### 빌드 방법

#### Make 사용
```bash
# 전체 빌드
make all

# CLI만 빌드
make cli

# GUI만 빌드
make gui

# 모든 플랫폼용 빌드
make build-all
```

#### 빌드 스크립트 사용
```bash
# macOS/Linux
chmod +x build.sh
./build.sh

# Windows
build.bat
```

#### 직접 빌드
```bash
# CLI 빌드
go build -ldflags="-s -w" -o bmw-monitor-cli cmd/cli/main.go

# GUI 빌드
go build -ldflags="-s -w" -o bmw-monitor-gui cmd/gui/*.go
```

## 사용 가능한 프로그램 목록 📋

### Experience Programs
- Test Drive (테스트 드라이브)
- Off-Road (오프로드)
- Taxi (택시)
- i Drive (i 드라이브)
- Night Drive (나이트 드라이브)
- Scenic Drive (시닉 드라이브)
- On-Road (온로드)
- X-Bus (X-버스)

### Training Programs
- Starter Pack (스타터 팩)
- i Starter Pack (i 스타터 팩)
- MINI Starter Pack (MINI 스타터 팩)
- M Core (M 코어)
- BEV Core (BEV 코어)
- Intensive (인텐시브)
- M Intensive (M 인텐시브)
- JCW Intensive (JCW 인텐시브)
- M Drift I (M 드리프트 I)
- M Drift II (M 드리프트 II)
- M Drift III (M 드리프트 III)

### Owner Programs
- Owners Track Day (오너스 트랙 데이)
- Owners Drift Day (오너스 드리프트 데이)

### Junior Campus Programs
- Laboratory (연구실)
- Workshop (워크샵)

## Gmail 앱 비밀번호 설정 📧

1. Google 계정 설정 → 보안
2. 2단계 인증 활성화
3. 앱 비밀번호 생성
4. 생성된 16자리 비밀번호를 config.yaml에 입력

## 프로젝트 구조 📁

```
bmw-driving-center-alter/
├── cmd/
│   ├── cli/          # CLI 프로그램
│   └── gui/          # GUI 프로그램
├── internal/
│   ├── browser/      # 브라우저 자동화
│   ├── config/       # 설정 관리
│   ├── models/       # 데이터 모델
│   └── notifier/     # 이메일 알림
├── configs/
│   └── config.yaml   # 설정 파일
├── build/            # 빌드된 실행 파일
├── build.sh          # macOS/Linux 빌드 스크립트
├── build.bat         # Windows 빌드 스크립트
└── Makefile          # Make 빌드 설정
```

## 주의사항 ⚠️

1. **세션 유지**: 반복적인 로그인은 캡챠를 유발할 수 있으므로 세션이 자동으로 저장됩니다.
   - 세션은 `~/.bmw-driving-center/browser-state/`에 저장됩니다.

2. **브라우저 설치**: Playwright 브라우저는 한 번만 설치하면 됩니다.

3. **설정 저장**: GUI에서 선택한 프로그램은 자동으로 저장되어 다음 실행 시 복원됩니다.

4. **이메일 설정**: Gmail의 경우 앱 비밀번호가 필요합니다.

## 문제 해결 🔧

### 로그인 실패
- BMW ID와 비밀번호를 다시 확인하세요
- 세션 파일을 삭제하고 다시 시도: `rm -rf ~/.bmw-driving-center/browser-state/`

### 브라우저 오류
- Playwright 브라우저 재설치: 
  ```bash
  go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
  ```

### 이메일 전송 실패
- Gmail 앱 비밀번호를 확인하세요
- SMTP 설정이 올바른지 확인하세요
- 방화벽에서 SMTP 포트(587)가 열려있는지 확인하세요

### GUI 실행 오류 (macOS)
- 보안 설정에서 앱 실행 허용이 필요할 수 있습니다
- 시스템 환경설정 → 보안 및 개인 정보 보호 → 일반 → "확인 없이 열기" 클릭

## 개발 환경 설정 🛠️

```bash
# 저장소 클론
git clone https://github.com/yourusername/bmw-driving-center-alter.git
cd bmw-driving-center-alter

# 의존성 설치
go mod download

# 개발 실행
make run-gui  # GUI 실행
make run-cli  # CLI 실행

# 테스트
make test
```

## 기여하기 🤝

Pull Request와 Issue를 환영합니다!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 라이선스 📄

MIT License - 자유롭게 사용하세요!

## 감사의 말 💙

BMW 드라이빙 센터의 멋진 프로그램들을 더 많은 분들이 경험할 수 있기를 바랍니다.

---
Made with ❤️ for BMW Driving Experience