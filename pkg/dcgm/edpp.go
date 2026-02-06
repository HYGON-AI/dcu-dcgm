package dcgm

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// -------------------- 常量定义 --------------------

// EDPPLogDir 定义 EDPp 日志目录（相对可执行文件目录）
const EDPPLogDir = "logs/edpp"

// EDPpStressTime 定义每个模式的测试时间（秒）
const EDPpStressTime = 10

// -------------------- 数据结构 --------------------

// ErrorSummary 保存每个 DCU 的硬件错误统计（原有类型，文件中已有使用）
// 保留：用于内部复用 checkHardwareError 返回值
type ErrorSummary struct {
	DCU        int
	ECCErr     int
	MemErr     int
	ComputeErr int
}

// DCUStatus 用于记录 DCU 的实时采集信息（保留原有定义）
type DCUStatus struct {
	Utilization string
	Power       string
	GFXClock    string
	Temperature string
}

// -------------------- EDPp 可执行文件嵌入 --------------------

//go:embed resources/EDPp
var edppBytes []byte

// extractEDPp 将嵌入的 EDPp 二进制写入临时文件，并返回可执行路径
func extractEDPp() (string, error) {
	if len(edppBytes) == 0 {
		return "", fmt.Errorf("embedded EDPp binary is empty")
	}

	tmpFile, err := os.CreateTemp("", "EDPp-*")
	if err != nil {
		return "", fmt.Errorf("无法创建临时文件: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(edppBytes); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("无法设置执行权限: %w", err)
	}

	return tmpFile.Name(), nil
}

// -------------------- 硬件错误统计 --------------------

// checkHardwareError 按模式名读取每个 DCU 日志并统计错误（原有函数，保留不变）
func checkHardwareError(logDir string, dcus []int, pattern string) ([]ErrorSummary, error) {
	var results []ErrorSummary
	for _, dcu := range dcus {
		logFile := filepath.Join(logDir, fmt.Sprintf("edpp%s_dcu%d.log", pattern, dcu))
		data, err := os.ReadFile(logFile)
		if err != nil {
			return nil, fmt.Errorf("无法读取日志 %s: %w", logFile, err)
		}

		content := string(data)
		summary := ErrorSummary{DCU: dcu}
		summary.ECCErr = strings.Count(content, "ECC error")
		summary.MemErr = strings.Count(content, "Memory error")
		summary.ComputeErr = strings.Count(content, "Compute error")

		results = append(results, summary)
	}
	return results, nil
}

// -------------------- 实时监控 goroutine --------------------

// monitorEdpp 采集指定 DCU 的 DCU 信息并写入日志文件（原函数，保留不变）
func monitorEdpp(logFile string, totalHCU int, stop chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法打开日志文件 %s: %v\n", logFile, err)
		return
	}
	defer f.Close()

	// 写入 CSV 表头
	fmt.Fprintln(f, "index,timestamp,utilization.dcu [%],power.draw [W],clocks.current.gfx [MHz],temperature.dcu")

	for {
		select {
		case <-stop:
			return
		default:
			for i := 0; i < totalHCU; i++ {
				now := time.Now()
				timestamp := now.Format("2006-01-02 15:04:05")

				util, power, temp, gfx := "0", "0", "0", "0"

				// hy-smi 命令获取利用率、功耗、温度
				cmdTemp := exec.Command("sh", "-c",
					fmt.Sprintf("hy-smi -d %d | grep 'Normal' | awk '{print $7, $3, $2}'", i))
				out, _ := cmdTemp.Output()
				fields := strings.Fields(string(out))
				if len(fields) >= 3 {
					util, power, temp = fields[0], fields[1], fields[2]
				}

				// hy-smi 命令获取 GFX Clock
				cmdGfx := exec.Command("sh", "-c",
					fmt.Sprintf("hy-smi -d %d -c | grep sclk | awk -F'[()]' '{print $2}'", i))
				out, _ = cmdGfx.Output()
				if len(out) > 0 {
					gfx = strings.TrimSpace(string(out))
				}

				fmt.Fprintf(f, "%d,%s,%s,%s,%s,%s\n", i, timestamp, util, power, gfx, temp)
			}
			time.Sleep(1 * time.Second) // 采集间隔
		}
	}
}

// -------------------- 测试模式列表 --------------------

var edppNames = []string{"10KHz", "5KHz", "4KHz", "3KHz", "2KHz", "1.5KHz", "1KHz", "800Hz", "500Hz", "200Hz", "100Hz", "50Hz", "100ms"}

// -------------------- EDPp 测试主函数（原有、不变） --------------------

// RunEDPpTest 主函数，运行指定 DCU 列表的 EDPp 测试
// dvIdList: 要测试的 DCU ID 列表
func edpppTest() {
	// 获取 DCU 总数
	totalHCU, _ := rsmiNumMonitorDevices()
	if totalHCU <= 0 {
		fmt.Printf("获取 DCU 总数失败")
		return
	}
	dvIdList := make([]int, totalHCU)
	for i := 0; i < totalHCU; i++ {
		dvIdList[i] = i
	}

	// 提取可执行文件
	edppPath, err := extractEDPp()
	if err != nil {
		fmt.Printf("提取 EDPp 失败: %v\n", err)
		return
	}
	defer os.Remove(edppPath)

	// 创建日志目录
	if err := os.MkdirAll(EDPPLogDir, 0755); err != nil {
		fmt.Printf("无法创建日志目录: %v\n", err)
		return
	}

	// 初始化 summary 用于汇总每个 DCU 的错误信息
	summary := make([]map[string]string, totalHCU)
	for i := 0; i < totalHCU; i++ {
		summary[i] = make(map[string]string)
	}

	// 遍历所有测试模式
	for i, name := range edppNames {
		fmt.Printf("===== Edpp Test Pattern: %s start =====\n", name)
		for _, dcu := range dvIdList {
			logFile := filepath.Join(EDPPLogDir, fmt.Sprintf("edpp%s_dcu%d.log", name, dcu))
			stop := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(1)
			go monitorEdpp(logFile, 1, stop, &wg) // 单个 DCU 监控

			// 执行 EDPp 测试
			cmd := exec.Command(edppPath, "-t", fmt.Sprintf("%d", EDPpStressTime), "-f", fmt.Sprintf("%d", i))
			cmd.Stdout = nil
			cmd.Stderr = nil
			_ = cmd.Run()

			close(stop)
			wg.Wait()

			// 初始化 summary 索引对应 DCU
			//summary[idx] = make(map[string]string)
		}

		// 统计硬件错误
		errors, err := checkHardwareError(EDPPLogDir, dvIdList, name)
		if err != nil {
			fmt.Printf("检查硬件错误失败: %v\n", err)
		} else {
			fmt.Printf("----- Hardware Error Summary for pattern %s -----\n", name)
			for idx, e := range errors {
				fmt.Printf("DCU %d -> ECC: %d, Mem: %d, Compute: %d\n", e.DCU, e.ECCErr, e.MemErr, e.ComputeErr)
				summary[idx][name+"_ECC"] = fmt.Sprintf("%d", e.ECCErr)
				summary[idx][name+"_Mem"] = fmt.Sprintf("%d", e.MemErr)
				summary[idx][name+"_Compute"] = fmt.Sprintf("%d", e.ComputeErr)
			}
			fmt.Println(strings.Repeat("-", 50))
		}

		fmt.Printf("===== Edpp Test Pattern: %s end =====\n\n", name)
	}

	// 打印最终 summary 表格
	fmt.Println("===== EDPp Test Final Summary =====")
	for idx, dcuSummary := range summary {
		fmt.Printf("[DCU %d]\n", dvIdList[idx])
		fmt.Printf("%-35s %-10s\n", "Test Item", "Error Count")
		fmt.Println(strings.Repeat("-", 50))
		for test, errCount := range dcuSummary {
			fmt.Printf("%-35s %-10s\n", test, errCount)
		}
		fmt.Println()
	}
}

// -------------------- 新增：结构化 API（保留原实现不变） --------------------

// PatternResult 表示某个测试模式下某个 DCU 的错误统计（字段名规范）
type PatternResult struct {
	PatternName       string // 测试模式名称（例如 "10KHz", "5KHz" 等）
	ECCCount          int    // 该模式下检测到的 ECC 错误数
	MemoryErrorCount  int    // 该模式下检测到的内存错误数
	ComputeErrorCount int    // 该模式下检测到的计算相关错误数
}

// DCUEdppResult 表示某个 DCU 在所有测试模式下的统计结果
type DCUEdppResult struct {
	DCUId          int             // DCU 索引（设备编号）
	PatternResults []PatternResult // 该 DCU 在各个测试模式下的错误统计列表（按测试顺序或名称顺序）
}

// EDPPResult 汇总整个 EDPp 测试的结构化结果
type EDPPResult struct {
	DCUEdppResults []DCUEdppResult // 每个 DCU 的汇总结果列表
	LogDir         string          // 存放生成的日志文件的目录路径
}

// runEdppTestWithResult 内部实现：复现 edpppTest 的流程，但返回结构化数据（不打印） // NEW
func runEdppTestWithResult() (EDPPResult, error) {
	var res EDPPResult

	// 获取 DCU 总数
	totalHCU, _ := rsmiNumMonitorDevices()
	if totalHCU <= 0 {
		return res, fmt.Errorf("获取 DCU 总数失败")
	}
	dvIdList := make([]int, totalHCU)
	for i := 0; i < totalHCU; i++ {
		dvIdList[i] = i
	}

	// 提取可执行文件
	edppPath, err := extractEDPp()
	if err != nil {
		return res, fmt.Errorf("提取 EDPp 失败: %w", err)
	}
	defer os.Remove(edppPath)

	// 创建日志目录
	if err := os.MkdirAll(EDPPLogDir, 0755); err != nil {
		return res, fmt.Errorf("无法创建日志目录: %w", err)
	}
	res.LogDir = EDPPLogDir

	// 使用中间结构先收集每个 DCU 的模式结果（map DCU -> []PatternResult）
	dcuMap := make(map[int][]PatternResult)
	for _, d := range dvIdList {
		dcuMap[d] = make([]PatternResult, 0, len(edppNames))
	}

	// 遍历所有测试模式（与原 edpppTest 保持一致的执行流程）
	for i, name := range edppNames {
		for _, dcu := range dvIdList {
			logFile := filepath.Join(EDPPLogDir, fmt.Sprintf("edpp%s_dcu%d.log", name, dcu))
			stop := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(1)
			// 与原逻辑一致：对单个 DCU 监控，传入 totalHCU=1
			go monitorEdpp(logFile, 1, stop, &wg)

			// 执行 EDPp 测试（与原逻辑一致）
			cmd := exec.Command(edppPath, "-t", fmt.Sprintf("%d", EDPpStressTime), "-f", fmt.Sprintf("%d", i))
			cmd.Stdout = nil
			cmd.Stderr = nil
			_ = cmd.Run()

			close(stop)
			wg.Wait()
		}

		// 在每个模式完成后统计硬件错误（复用原有 checkHardwareError）
		errors, err := checkHardwareError(EDPPLogDir, dvIdList, name)
		if err != nil {
			// 如果统计失败，继续下一个模式并记录为 0
			for _, dcu := range dvIdList {
				dcuMap[dcu] = append(dcuMap[dcu], PatternResult{
					PatternName:       name,
					ECCCount:          0,
					MemoryErrorCount:  0,
					ComputeErrorCount: 0,
				})
			}
			continue
		}

		// 将 checkHardwareError 的结果（ErrorSummary）转换为 PatternResult 并收集
		for _, e := range errors {
			pr := PatternResult{
				PatternName:       name,
				ECCCount:          e.ECCErr,
				MemoryErrorCount:  e.MemErr,
				ComputeErrorCount: e.ComputeErr,
			}
			dcuMap[e.DCU] = append(dcuMap[e.DCU], pr)
		}
	}

	// 把 map 转成 []DCUEdppResult（保持 dcu 索引顺序）
	results := make([]DCUEdppResult, 0, len(dvIdList))
	for _, d := range dvIdList {
		dr := DCUEdppResult{
			DCUId:          d,
			PatternResults: dcuMap[d],
		}
		results = append(results, dr)
	}
	res.DCUEdppResults = results

	return res, nil
}
