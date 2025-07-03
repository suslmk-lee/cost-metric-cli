# K-PaaS Cost - Kubernetes CSP Cost Analysis CLI

K-PaaS Cost는 Kubernetes 클러스터의 CSP(Cloud Service Provider) 비용을 분석하는 CLI 도구입니다.

## 특징

- 🔍 **자동 CSP 감지**: Kubernetes 컨텍스트를 기반으로 CSP 자동 식별
- 📊 **실시간 비용 계산**: CSP API를 통한 정확한 인스턴스 가격 조회
- 🖥️ **대화형 CLI**: 사용자 친화적인 대화형 인터페이스 제공
- 📈 **상세 리포트**: 노드별, 인스턴스 타입별 비용 분석 리포트
- 🌐 **멀티 CSP 지원**: Naver Cloud Platform, NHN Cloud 지원

## 지원 CSP

### 1단계 (완료)
- ✅ Naver Cloud Platform (NCP)

### 2단계 (예정)
- 🔄 NHN Cloud

### 향후 계획
- AWS EKS
- Azure AKS
- Google GKE

## 설치

### 직접 빌드
```bash
git clone https://github.com/minkyulee/cost-metric-cli.git
cd cost-metric-cli
go build -o bin/kpaas-cost ./cmd/kpaas-cost
```

### 바이너리 설치
```bash
# 바이너리를 PATH에 추가
mv bin/kubecon /usr/local/bin/
```

## 사용법

### 기본 명령어

```bash
# 대화형 모드 시작
kubecon

# 현재 컨텍스트 비용 분석
kubecon analyze

# 사용 가능한 컨텍스트 목록
kubecon list-contexts

# 가격 정보 조회
kubecon pricing

# 상세 비용 리포트 생성
kubecon report
```

### 플래그 옵션

```bash
# 특정 컨텍스트 지정
kubecon analyze --context my-cluster

# 계산 기간 설정
kubecon analyze --duration 24h

# 출력 형식 지정
kubecon analyze --format json

# 결과를 파일로 저장
kubecon report --output cost-report.json
```

### 대화형 모드

`kubecon` 명령어를 실행하면 대화형 메뉴가 표시됩니다:

```
========================================
           KUBECON v1.0
    Kubernetes Cost Analysis
========================================
1. 현재 컨텍스트 비용 분석
2. 컨텍스트 변경
3. 가격 정보 조회
4. 비용 리포트 생성
5. 설정
6. 종료
========================================
```

## 설정

### API 키 설정

CSP API 호출을 위한 인증 정보를 설정합니다:

```bash
# 환경 변수로 설정
export NAVER_ACCESS_KEY="your_access_key"
export NAVER_SECRET_KEY="your_secret_key"

export NHN_TOKEN="your_api_token"
```

### 설정 파일

`~/.kubecon.yaml` 파일을 생성하여 기본 설정을 지정할 수 있습니다:

```yaml
default_context: "my-cluster"
default_duration: "1h"
output_format: "table"
default_region: "KR-1"

naver:
  access_key: "your_access_key"
  secret_key: "your_secret_key"

nhn:
  token: "your_api_token"
```

## 아키텍처

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Interface │    │  Cost Calculator│    │   CSP API      │
│    (kubecon)    │───▶│     Engine      │───▶│   Clients      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Kubernetes     │    │  Instance Spec  │    │  Pricing Data   │
│   Context       │    │   Detection     │    │   Repository    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 개발

### 프로젝트 구조

```
cost-metric-cli/
├── cmd/
│   └── kubecon/           # CLI 진입점
├── pkg/
│   ├── k8s/              # Kubernetes 클라이언트
│   ├── csp/              # CSP 감지 및 클라이언트
│   ├── calculator/       # 비용 계산 엔진
│   └── cli/              # CLI 인터페이스
├── configs/              # 설정 파일
└── docs/                 # 문서
```

### 개발 명령어

```bash
# 의존성 설치
go mod tidy

# 빌드
go build -o bin/kubecon ./cmd/kubecon

# 테스트
go test ./...

# 린트
golangci-lint run
```

## 기여

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 라이선스

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 로드맵

- [x] 기본 CLI 구조 구현
- [x] Kubernetes 클라이언트 연동
- [x] CSP 자동 감지 기능
- [ ] Naver Cloud Platform API 연동
- [ ] 비용 계산 엔진
- [ ] 상세 리포트 생성
- [ ] NHN Cloud 지원
- [ ] 웹 대시보드 (향후)
- [ ] Prometheus 메트릭 연동 (향후)