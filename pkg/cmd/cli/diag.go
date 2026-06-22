/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HYGON-AI/dcu-dcgm/v2/pkg/dcgm"
)

// ==================== diag 命令 ====================
var diagCmd = &cobra.Command{
	Use:   "diag",
	Short: "Run diagnostics commands",
	Example: `  dcgmi diag -g <groupId> -i <flags>
  dcgmi diag r <diagLevel>
  dcgmi diag bandwidth <dcuId>
  dcgmi diag memtestCL <dcuId>
  dcgmi diag pcie 
  dcgmi diag xhcl
  dcgmi diag edpp
  dcgmi diag gemm`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case groupId != "":
			handleDiagGroup()
		case infoFlags != "":
			fmt.Println("Error: No group has been specified.")
		default:
			cmd.Help()
		}
	},
}

// ==================== diag run ====================
var runDiagCmd = &cobra.Command{
	Use:   "r [level]",
	Short: "Run general diagnostics. Usage: dcgmi diag r <level> (1~4)",
	Long: `Run a comprehensive diagnostic check on all devices with specified level.

Level 1: Basic checks
  - Memory Check
  - picBus Check
  - Power Check
  - DTK Version Check
  - Driver Version Check
  - RSMI Version Check
  - VBIOS Version Check
  - card Compatibility Check

Level 2: Level 1 + 
  - PCIe Bandwidth Check
  - Memory Bandwidth Check
  - Memory Reserved Pages Check

Level 3: Level 2 +
  - PCIe Bandwidth Stress Test
  - xHCL Bandwidth Test
  - Target Stress Test

Level 4: Level 3 +
  - MemtestCL
  - EDPpTest`,
	Args: cobra.ExactArgs(1), // 必须传一个参数
	Run: func(cmd *cobra.Command, args []string) {
		level, err := strconv.Atoi(args[0])
		if err != nil || level < 1 || level > 4 {
			fmt.Println("Invalid level. Please provide a value between 1 and 4.")
			os.Exit(1)
		}

		results, err := dcgm.RunDiag(level)
		if err != nil {
			fmt.Println("Error running diagnostics:", err)
			os.Exit(1)
		}
		printDiagnosticResults(results)
	},
}

// ==================== diag bandwidth ====================
var bandwidthCmd = &cobra.Command{
	Use:   "bandwidth [DCU IDs]",
	Short: "Run DCU bandwidth test",
	Long: `Run a DCU memory bandwidth stress test on specified devices.

If no device IDs are provided, the test will run on all available DCUs.

Usage:
  dcgmi diag bandwidth [DCU IDs]

Examples:
  dcgmi diag bandwidth
  dcgmi diag bandwidth 0
  dcgmi diag bandwidth 0 1 2

[DCU IDs] are the numeric IDs of the DCU devices you want to test.`,
	Run: func(cmd *cobra.Command, args []string) {
		var dvIdList []int

		if len(args) == 0 {
			// 如果没有输入 ID，则检查全部 DCU
			numDevices, err := dcgm.NumMonitorDevices()
			if err != nil {
				fmt.Println("Error retrieving device count:", err)
				os.Exit(1)
			}
			for i := 0; i < numDevices; i++ {
				dvIdList = append(dvIdList, i)
			}
		} else {
			// 把命令行参数转成 int slice
			for _, arg := range args {
				id, err := strconv.Atoi(arg)
				if err != nil {
					fmt.Printf("Invalid DCU ID '%s'\n", arg)
					os.Exit(1)
				}
				dvIdList = append(dvIdList, id)
			}
		}

		fmt.Printf("Running memory bandwidth test for DCU: %v\n", dvIdList)
		if !dcgm.BandwidthTest(dvIdList) {
			fmt.Println("Bandwidth test failed.")
			os.Exit(1)
		}
		//fmt.Println("Successfully completed memory bandwidth test.")
		fmt.Println("memory bandwidth test: Done.")
	},
}

// ==================== diag pcieBandwidth ====================
var pcieCmd = &cobra.Command{
	Use:   "pcie",
	Short: "Run PCIe memory bandwidth test",
	Long: `Run a PCIe memory bandwidth stress test on all dcu.

It evaluates bandwidth for Sys->Fb, Fb->Sys, and XHCL P2P transfers.

Usage:
  dcgmi diag pcie`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("=== Running PCIe Bandwidth Test ===")
		ok := dcgm.PcieBandwidthTest()
		if !ok {
			fmt.Println("PCIe bandwidth test failed.")
			os.Exit(1)
		}
		//fmt.Println("PCIe bandwidth test completed successfully.")
		fmt.Println("PCIe bandwidth test: Done.")
		//fmt.Printf("Logs directory: %s\n", dcgm.PCIELogDir)
	},
}

// ==================== diag xhcl ====================
var xhclCmd = &cobra.Command{
	Use:   "xhcl",
	Short: "Run XHCL stress test",
	Long: `Run an XHCL P2P bandwidth stress test across all dcu.

Usage:
  dcgmi diag xhcl`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running XHCL stress test on all dcu...")
		ok := dcgm.XHCLTest()
		if !ok {
			fmt.Printf("XHCL stress test failed: %v\n")
			os.Exit(1)
		}
		//fmt.Println("Successfully completed XHCL stress test.")
		fmt.Println("XHCL stress test: Done.")
	},
}

// ==================== diag edpp ====================
var edppCmd = &cobra.Command{
	Use:   "edpp",
	Short: "Run DCU EDPp stability test",
	Long: `Run the EDPp stress test on all DCU devices.

This test runs multiple frequency patterns to evaluate stability, monitor utilization,
power, temperature, and record error statistics.

Usage:
  dcgmi diag edpp

Examples:
  dcgmi diag edpp`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running EDPp test for all DCUs...")
		dcgm.EDPpTest()
		//fmt.Println("Successfully completed EDPp test.")
		fmt.Println("EDPp test: Done.")
	},
}

// ==================== diag gemm ====================
var gemmCmd = &cobra.Command{
	Use:   "gemm",
	Short: "Run GEMM target stress test",
	Long: `Run a GEMM target stress test on all dcu.

This test stresses compute performance using GEMM workloads.

Usage:
  dcgmi diag gemm`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running GEMM target stress test on all dcu...")
		dcgm.TargetStressTest()
		//fmt.Println("Successfully completed GEMM target stress test.")
		fmt.Println("GEMM stress test: Done.")
	},
}

// ==================== diag memtestCL ====================
var memtestCLCmd = &cobra.Command{
	Use:   "memtestCL [DCU IDs]",
	Short: "Run DCU memory stress test",
	Long: `Run a memory stress and integrity test on specified DCU devices.

If no device IDs are provided, the test will run on all available DCUs.
It detects memory errors under heavy load and produces detailed logs.

Usage:
  dcgmi diag memtestCL [DCU IDs]

Examples:
  dcgmi diag memtestCL
  dcgmi diag memtestCL 0
  dcgmi diag memtestCL 0 1 2`,
	Run: func(cmd *cobra.Command, args []string) {
		var dvIdList []int

		if len(args) == 0 {
			// 如果没有输入 ID，则检查全部 DCU
			numDevices, err := dcgm.NumMonitorDevices()
			if err != nil {
				fmt.Println("Error retrieving device count:", err)
				os.Exit(1)
			}
			for i := 0; i < numDevices; i++ {
				dvIdList = append(dvIdList, i)
			}
		} else {
			// 把命令行参数转成 int slice
			for _, arg := range args {
				id, err := strconv.Atoi(arg)
				if err != nil {
					fmt.Printf("Invalid DCU ID '%s'\n", arg)
					os.Exit(1)
				}
				dvIdList = append(dvIdList, id)
			}
		}

		fmt.Printf("Running memtestCL stress test for DCU(s): %v\n", dvIdList)
		if err := dcgm.MemtestCL(dvIdList); err != nil {
			fmt.Printf("MemtestCL test failed: %v\n", err)
			os.Exit(1)
		}
		//fmt.Println("Successfully completed memtestCL stress test.")
		fmt.Println("memtestCL stress test: Done.")
	},
}

// -------------------- 辅助打印函数 --------------------
func printDiagnosticResults(results dcgm.DiagResults) {
	fmt.Println("Successfully ran diagnostic for DCU.")
	fmt.Println("+---------------------------+------------------------------------------------+")
	fmt.Println("| Diagnostic                | Result                                         |")
	fmt.Println("+===========================+================================================+")

	// 输出 Software 部分
	if len(results.Software) > 0 {
		fmt.Println("| Software Diagnostics      |                                                |")
		fmt.Println("+---------------------------+------------------------------------------------+")
		for _, result := range results.Software {
			printTestResult(result)
		}
	}

	// 输出 PerDCU 部分
	if len(results.PerDCU) > 0 {
		fmt.Println("| Hardware Diagnostics      |                                                |")
		fmt.Println("+---------------------------+------------------------------------------------+")
		for _, dcu := range results.PerDCU {
			fmt.Printf("| DCU: %d                     |------------------------------------------------|\n", dcu.DCU)
			for _, result := range dcu.DiagResults {
				printTestResult(result)
			}
		}
	}

	fmt.Println("+---------------------------+------------------------------------------------+")
}

func printTestResult(result dcgm.DiagResult) {
	fmt.Printf("| %-25s | %-46s |\n", result.TestName, result.Status)
	if result.ErrorMessage != "" {
		fmt.Printf("| %-25s | %-46s |\n", "Error Message", result.ErrorMessage)
	}
	if result.TestOutput != "" {
		fmt.Printf("| %-25s | %-46s |\n", "Test Output", result.TestOutput)
	}
	if result.ErrorCode != 0 {
		fmt.Printf("| %-25s | %-46d |\n", "Error Code", result.ErrorCode)
	}
	fmt.Println("+------------------------------------------------+")
}

// -------------------- 初始化 --------------------
func init() {
	diagCmd.AddCommand(runDiagCmd)
	diagCmd.AddCommand(bandwidthCmd)
	diagCmd.AddCommand(pcieCmd)
	rootCmd.AddCommand(diagCmd)
	diagCmd.AddCommand(xhclCmd)
	diagCmd.AddCommand(gemmCmd)
	diagCmd.AddCommand(memtestCLCmd)
	diagCmd.AddCommand(edppCmd)

	diagCmd.Flags().StringVarP(&groupId, "group", "g", "", "The group ID to query.")
	diagCmd.Flags().StringVarP(&infoFlags, "info", "i", "", "Specify which information to return.\n"+
		" b - memory bandwidth\n m - memtestCL stress")
}

func handleDiagGroup() {
	if infoFlags == "" {
		fmt.Println("Error: No info flag has been specified.")
		return
	}
	for _, c := range infoFlags {
		switch c {
		case 'b', 'm':
		default:
			fmt.Printf("Invalid input '%c'. Please include only valid tags.\n", c)
			return
		}
	}
	groupIdInt, err := strconv.Atoi(groupId)
	if err != nil {
		fmt.Println("Error: Invalid Group ID given.")
		return
	}
	dcuInGroup, _, err := dcgm.GetDcuListFromGroup(groupIdInt)
	if err != nil {
		fmt.Printf("Error getting group info: %v\n", err)
		return
	}
	if len(dcuInGroup) == 0 {
		fmt.Printf("Failed to query group: no entity found for group %v\n", groupId)
		return
	}
	if strings.Contains(infoFlags, "b") {
		fmt.Printf("Running memory bandwidth test for DCU(s): %v\n", dcuInGroup)
		if !dcgm.BandwidthTest(dcuInGroup) {
			fmt.Println("Bandwidth test failed.")
			return
		}
		fmt.Println("Successfully completed memory bandwidth test.")
	}
	if strings.Contains(infoFlags, "m") {
		fmt.Printf("Running memtestCL stress test for DCU(s): %v\n", dcuIndex)
		if err := dcgm.MemtestCL(dcuInGroup); err != nil {
			fmt.Printf("MemtestCL test failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully completed memtestCL stress test ✅")
	}
}
