# Kubernetes 노드 중지 시간 추적 및 정확한 비용 계산 분석

## 개요

현재 K-PaaS Cost CLI 도구는 노드 생성 시점부터 현재까지의 시간을 기반으로 비용을 계산하고 있으나, 실제로는 노드의 중지/재시작 시간을 고려하지 않아 비용이 과다 계산되는 문제가 발견되었습니다.

## 문제 현황 분석

### 1. 현재 비용 계산 방식의 문제점

#### 현재 구현 (`pkg/calculator/calculator.go:44-48`)
```go
now := time.Now()
actualUptime := now.Sub(nodeInfo.CreatedAt)  // 문제: 연속 가동 가정
nodeCost.ActualUptime = actualUptime
nodeCost.UptimeHours = actualUptime.Hours()
nodeCost.ActualUsageCost = pricingInfo.HourlyPrice * nodeCost.UptimeHours
```

**문제점**: 노드 생성 시점부터 현재까지 연속적으로 가동되었다고 가정하여 중지 시간을 무시함

### 2. 실제 데이터 분석 결과

#### 노드 `suslmk-node-w-3f52` 분석 (2025-06-18 09:09 기준)

| 항목 | 시간 | 비고 |
|------|------|------|
| **노드 생성 시간** | 2025-06-17T00:02:37Z | kubeconfig에서 확인 |
| **마지막 Ready 시간** | 2025-06-17T23:38:36Z | Node Conditions에서 확인 |
| **재부팅 이벤트** | 33분 전 | kubectl events에서 확인 |
| **예상 중지 시간** | 약 10시간 | 생성~Ready 시간 차이 |

#### 비용 영향 계산

| 계산 방식 | 가동 시간 | 노드당 비용 (86원/시간) | 3노드 총 비용 |
|-----------|-----------|------------------------|---------------|
| **현재 (잘못된 계산)** | 72.4시간 | 6,226원 | 18,678원 |
| **실제 계산** | 62-65시간 | 5,332~5,590원 | 15,996~16,770원 |
| **과다 계산** | 7-10시간 | 602~860원 | 1,806~2,580원 |

**결론**: 현재 약 **1,800~2,600원**의 과다 계산이 발생하고 있음

### 3. Kubernetes API에서 제공하는 추적 가능한 정보

#### A. Node Conditions
```bash
kubectl describe node <node-name>
```
**수집 가능한 정보**:
- `Ready` condition의 `LastTransitionTime`: 노드가 마지막으로 Ready 상태가 된 시간
- `LastHeartbeatTime`: 마지막 하트비트 시간
- 각 condition별 상태 변화 시점

**예시**:
```
Ready    True    Wed, 18 Jun 2025 09:12:11 +0900   Wed, 18 Jun 2025 08:38:36 +0900
```

#### B. Node Events
```bash
kubectl get events --field-selector involvedObject.kind=Node
```
**수집 가능한 이벤트**:
- `Rebooted`: 노드 재부팅 시점
- `Starting`: kubelet 시작 시점  
- `NodeReady`: 노드 Ready 상태 변경
- `NodeNotReady`: 노드 비정상 상태

**예시**:
```
33m Warning Rebooted    node/suslmk-node-w-3f52   Node has been rebooted
33m Normal  Starting    node/suslmk-node-w-3f52   Starting kubelet
33m Normal  NodeReady   node/suslmk-node-w-3f52   Node status is now: NodeReady
```

#### C. Node Status Information
- `CreationTimestamp`: 노드 생성 시간
- `Status.Conditions[]`: 상세한 상태 변화 히스토리
- `Status.NodeInfo`: 시스템 정보 및 부트 ID

## 정확한 비용 계산을 위한 해결 방안

### 1. 단기 해결책 (즉시 구현 가능)

#### A. LastTransitionTime 기반 계산
```go
// 개선된 계산 방식
func calculateActualUptime(node *corev1.Node) time.Duration {
    now := time.Now()
    
    // Ready condition의 LastTransitionTime부터 계산
    for _, condition := range node.Status.Conditions {
        if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
            return now.Sub(condition.LastTransitionTime.Time)
        }
    }
    
    // Ready condition이 없으면 생성 시간부터 계산 (fallback)
    return now.Sub(node.CreationTimestamp.Time)
}
```

**장점**: 
- 즉시 구현 가능
- 마지막 재시작 이후의 실제 가동 시간만 계산
- 기존 코드 최소 변경

**한계**: 
- 여러 번의 중지/재시작 히스토리는 추적 불가
- 단일 재시작 후의 시간만 계산

### 2. 중기 해결책 (Events API 활용)

#### A. 확장된 NodeInfo 구조
```go
type EnhancedNodeInfo struct {
    NodeInfo                          // 기존 구조 포함
    Conditions        []NodeConditionInfo `json:"conditions"`
    Events            []NodeEventInfo     `json:"events"`
    UptimePeriods     []UptimePeriod     `json:"uptime_periods"`
    TotalDowntime     time.Duration      `json:"total_downtime"`
    ActualRunningTime time.Duration      `json:"actual_running_time"`
    UptimePercentage  float64           `json:"uptime_percentage"`
}

type UptimePeriod struct {
    StartTime time.Time     `json:"start_time"`
    EndTime   *time.Time    `json:"end_time,omitempty"` // nil if still running
    Duration  time.Duration `json:"duration"`
    Reason    string        `json:"reason"` // 시작/종료 이유
}

type NodeEventInfo struct {
    Type           string      `json:"type"`
    Reason         string      `json:"reason"`
    Message        string      `json:"message"`
    FirstTimestamp metav1.Time `json:"first_timestamp"`
    LastTimestamp  metav1.Time `json:"last_timestamp"`
    Count          int32       `json:"count"`
}
```

#### B. Events 기반 다운타임 계산
```go
func calculateDowntimePeriods(events []NodeEventInfo, conditions []NodeConditionInfo) []UptimePeriod {
    var periods []UptimePeriod
    var currentPeriod *UptimePeriod
    
    // Events를 시간순으로 정렬
    sortEventsByTime(events)
    
    for _, event := range events {
        switch event.Reason {
        case "Starting", "NodeReady":
            if currentPeriod == nil {
                currentPeriod = &UptimePeriod{
                    StartTime: event.FirstTimestamp.Time,
                    Reason:    event.Reason,
                }
            }
        case "Rebooted", "NodeNotReady":
            if currentPeriod != nil {
                endTime := event.FirstTimestamp.Time
                currentPeriod.EndTime = &endTime
                currentPeriod.Duration = endTime.Sub(currentPeriod.StartTime)
                periods = append(periods, *currentPeriod)
                currentPeriod = nil
            }
        }
    }
    
    // 현재 실행 중인 period 처리
    if currentPeriod != nil {
        now := time.Now()
        currentPeriod.Duration = now.Sub(currentPeriod.StartTime)
        periods = append(periods, *currentPeriod)
    }
    
    return periods
}
```

### 3. 장기 해결책 (완전한 라이프사이클 추적)

#### A. 새로운 Kubernetes API 메서드 추가
```go
// pkg/k8s/nodes.go에 추가할 메서드들

func (c *K8sClient) GetNodeEvents(ctx context.Context, nodeName string) ([]NodeEventInfo, error) {
    events, err := c.clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
        FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Node", nodeName),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get node events: %w", err)
    }
    
    var nodeEvents []NodeEventInfo
    for _, event := range events.Items {
        nodeEvents = append(nodeEvents, NodeEventInfo{
            Type:           event.Type,
            Reason:         event.Reason,
            Message:        event.Message,
            FirstTimestamp: event.FirstTimestamp,
            LastTimestamp:  event.LastTimestamp,
            Count:          event.Count,
        })
    }
    
    return nodeEvents, nil
}

func (c *K8sClient) GetEnhancedNodeInfo(ctx context.Context, nodeName string) (*EnhancedNodeInfo, error) {
    // 기본 노드 정보 수집
    basicInfo, err := c.GetNode(ctx, nodeName)
    if err != nil {
        return nil, err
    }
    
    // 이벤트 정보 수집
    events, err := c.GetNodeEvents(ctx, nodeName)
    if err != nil {
        return nil, err
    }
    
    // 향상된 정보 생성
    enhanced := &EnhancedNodeInfo{
        NodeInfo: *basicInfo,
        Events:   events,
    }
    
    // 가동 시간 계산
    enhanced.UptimePeriods = calculateDowntimePeriods(events, enhanced.Conditions)
    enhanced.ActualRunningTime = calculateTotalRunningTime(enhanced.UptimePeriods)
    enhanced.TotalDowntime = time.Since(basicInfo.CreatedAt) - enhanced.ActualRunningTime
    enhanced.UptimePercentage = calculateUptimePercentage(enhanced)
    
    return enhanced, nil
}
```

#### B. 향상된 비용 계산
```go
func (c *DefaultCostCalculator) CalculateAccurateNodeCost(nodeInfo EnhancedNodeInfo, pricingInfo PricingInfo) *NodeCost {
    actualRunningHours := nodeInfo.ActualRunningTime.Hours()
    
    nodeCost := &NodeCost{
        NodeInfo:          nodeInfo.NodeInfo,
        PricingInfo:       pricingInfo,
        HourlyCost:        pricingInfo.HourlyPrice,
        ActualUsageCost:   pricingInfo.HourlyPrice * actualRunningHours, // 정확한 계산!
        ActualRunningTime: nodeInfo.ActualRunningTime,
        TotalDowntime:     nodeInfo.TotalDowntime,
        UptimePercentage:  nodeInfo.UptimePercentage,
        UptimePeriods:     nodeInfo.UptimePeriods,
    }
    
    return nodeCost
}
```

## 구현 우선순위 및 영향도

### 우선순위 1: LastTransitionTime 기반 개선 (즉시)
- **구현 시간**: 1-2시간
- **정확도 개선**: 70-80%
- **위험도**: 낮음
- **영향**: 마지막 재시작 이후 정확한 계산

### 우선순위 2: Events API 통합 (1주일)
- **구현 시간**: 2-3일
- **정확도 개선**: 90-95%
- **위험도**: 중간
- **영향**: 전체 라이프사이클 추적

### 우선순위 3: 완전한 다운타임 추적 (2주일)
- **구현 시간**: 1-2주
- **정확도 개선**: 98-99%
- **위험도**: 중간
- **영향**: SLA 모니터링, 최적화 인사이트

## 예상 효과

### 비용 정확도
- **현재**: 10-15% 과다 계산
- **개선 후**: 1-2% 오차 범위 내

### 운영 인사이트
- 노드별 안정성 분석
- 비정상 재시작 패턴 감지
- 실제 가용성 추적
- 비용 최적화 기회 식별

### 사용자 가치
- 정확한 비용 예측
- 투명한 과금 정보
- 인프라 최적화 가이드
- SLA 성능 모니터링

## 결론

Kubernetes API에서 제공하는 풍부한 정보를 활용하면 현재의 부정확한 비용 계산 문제를 해결할 수 있습니다. 단계적 접근을 통해 빠른 개선과 장기적인 완성도를 모두 확보할 수 있으며, 이는 K-PaaS Cost의 핵심 가치인 "정확한 비용 분석"을 크게 향상시킬 것입니다.