package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func startInteractiveMode() {
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		showMainMenu()
		fmt.Print("선택하세요 (1-6): ")
		
		if !scanner.Scan() {
			break
		}
		
		choice := strings.TrimSpace(scanner.Text())
		
		switch choice {
		case "1":
			analyzeCurrentContext()
		case "2":
			switchContext()
		case "3":
			viewPricingInfo()
		case "4":
			generateCostReport()
		case "5":
			showSettings()
		case "6":
			fmt.Println("종료합니다.")
			os.Exit(0)
		default:
			fmt.Println("잘못된 선택입니다. 다시 입력해주세요.")
		}
		
		fmt.Println("\n계속하려면 Enter를 누르세요...")
		scanner.Scan()
	}
}

func showMainMenu() {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("         K-PaaS Cost")
	fmt.Println("    Kubernetes Cost Analysis")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println("1. 현재 컨텍스트 비용 분석")
	fmt.Println("2. 컨텍스트 변경")
	fmt.Println("3. 가격 정보 조회")
	fmt.Println("4. 비용 리포트 생성")
	fmt.Println("5. 설정")
	fmt.Println("6. 종료")
	fmt.Println(strings.Repeat("=", 40))
}

func analyzeCurrentContext() {
	fmt.Println("\n📊 현재 컨텍스트 비용 분석 중...")
	
	// Run cost analysis using interactive mode function
	if err := runCostAnalysisInteractive(); err != nil {
		fmt.Printf("❌ 분석 실패: %v\n", err)
		return
	}
}

func switchContext() {
	fmt.Println("\n🔄 컨텍스트 변경")
	
	// List available contexts
	if err := runListContextsInteractive(); err != nil {
		fmt.Printf("❌ 컨텍스트 목록 조회 실패: %v\n", err)
		return
	}
	
	// Get user choice
	choice := getUserInput("\n사용할 컨텍스트 번호를 입력하세요 (Enter로 취소): ")
	if choice == "" {
		fmt.Println("취소되었습니다.")
		return
	}
	
	fmt.Printf("컨텍스트 변경은 kubeconfig를 직접 수정해야 합니다.\n")
	fmt.Printf("다음 명령어를 사용하세요: kubectl config use-context <context-name>\n")
}

func viewPricingInfo() {
	fmt.Println("\n💰 가격 정보 조회")
	
	// Show pricing information using interactive mode function
	if err := runShowPricingInteractive(); err != nil {
		fmt.Printf("❌ 가격 정보 조회 실패: %v\n", err)
		return
	}
}

func generateCostReport() {
	fmt.Println("\n📋 비용 리포트 생성")
	
	// Get output format choice
	fmt.Println("출력 형식을 선택하세요:")
	fmt.Println("1. JSON")
	fmt.Println("2. YAML")
	
	choice := getUserChoice("선택 (1-2): ", 1, 2)
	
	format := "json"
	if choice == 2 {
		format = "yaml"
	}
	
	// Get output filename
	filename := getUserInput("출력 파일명 (Enter로 기본값 사용): ")
	if filename == "" {
		filename = fmt.Sprintf("cost-report-%s.%s", time.Now().Format("20060102-150405"), format)
	}
	
	// Set format in viper for the report command
	viper.Set("format", format)
	viper.Set("output", filename)
	
	// Generate report using the same logic as the CLI command
	if err := runGenerateReport(nil); err != nil {
		fmt.Printf("❌ 리포트 생성 실패: %v\n", err)
		return
	}
}

func showSettings() {
	fmt.Println("\n⚙️  설정")
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Println("\n설정 메뉴:")
		fmt.Println("1. API 키 설정")
		fmt.Println("2. 기본 지역 설정")
		fmt.Println("3. 출력 형식 설정")
		fmt.Println("4. 뒤로 가기")
		fmt.Print("선택하세요 (1-4): ")
		
		if !scanner.Scan() {
			break
		}
		
		choice := strings.TrimSpace(scanner.Text())
		
		switch choice {
		case "1":
			configureAPIKeys()
		case "2":
			configureDefaultRegion()
		case "3":
			configureOutputFormat()
		case "4":
			return
		default:
			fmt.Println("잘못된 선택입니다.")
		}
	}
}

func configureAPIKeys() {
	fmt.Println("\n🔑 API 키 설정")
	fmt.Println("구현 예정: CSP API 키 설정 기능")
}

func configureDefaultRegion() {
	fmt.Println("\n🌍 기본 지역 설정")
	fmt.Println("구현 예정: 기본 지역 설정 기능")
}

func configureOutputFormat() {
	fmt.Println("\n📄 출력 형식 설정")
	fmt.Println("구현 예정: 출력 형식 설정 기능")
}

func getUserInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func getUserChoice(prompt string, min, max int) int {
	for {
		input := getUserInput(prompt)
		choice, err := strconv.Atoi(input)
		if err != nil || choice < min || choice > max {
			fmt.Printf("잘못된 입력입니다. %d부터 %d까지의 숫자를 입력해주세요.\n", min, max)
			continue
		}
		return choice
	}
}