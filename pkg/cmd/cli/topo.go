/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package cli

import (
	"fmt"
	"github.com/HYGON-AI/dcu-dcgm/v2/pkg/dcgm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

var topoCmd = &cobra.Command{
	Use:   "topo",
	Short: "DCU Topology",
	Long:  `policy -- Used to find the topology of DCUs on the system.`,
	Example: `  dcgmi topo
  dcgmi topo -g <groupId>
  dcgmi topo --dcuid <dcuId>`,
	Run: func(cmd *cobra.Command, args []string) {
		// Main dispatcher logic
		switch {
		case groupId != "":
			handleByGroup()
		case dcuIndex != "":
			handleByIndex()
		case groupId == "" && dcuIndex == "":
			handleAllDcu()
		default:
			cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(topoCmd)

	topoCmd.Flags().StringVarP(&groupId, "group", "g", "", "The group ID to query.")
	topoCmd.Flags().StringVar(&dcuIndex, "dcuid", "", "The DCU ID to query.")
}

func parseCpuRangeToSlice(s string) ([]int, error) {
	var cpus []int
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid CPU range format: %s", part)
			}
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid start of CPU range '%s': %w", rangeParts[0], err)
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid end of CPU range '%s': %w", rangeParts[1], err)
			}
			if start > end {
				return nil, fmt.Errorf("start of CPU range (%d) cannot be greater than end (%d)", start, end)
			}
			for i := start; i <= end; i++ {
				cpus = append(cpus, i)
			}
		} else {
			cpuID, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid CPU ID '%s': %w", part, err)
			}
			cpus = append(cpus, cpuID)
		}
	}

	uniqueCPUs := make(map[int]bool)
	var result []int
	for _, cpu := range cpus {
		if _, exists := uniqueCPUs[cpu]; !exists {
			uniqueCPUs[cpu] = true
			result = append(result, cpu)
		}
	}
	sort.Ints(result)

	return result, nil
}

func formatCpuSliceToRange(cpus []int) string {
	if len(cpus) == 0 {
		return ""
	}

	var ranges []string
	start := cpus[0]
	end := cpus[0]

	for i := 1; i < len(cpus); i++ {
		if cpus[i] == end+1 {
			end = cpus[i]
		} else {
			if start == end {
				ranges = append(ranges, strconv.Itoa(start))
			} else {
				ranges = append(ranges, fmt.Sprintf("%d-%d", start, end))
			}
			start = cpus[i]
			end = cpus[i]
		}
	}

	if start == end {
		ranges = append(ranges, strconv.Itoa(start))
	} else {
		ranges = append(ranges, fmt.Sprintf("%d-%d", start, end))
	}

	return strings.Join(ranges, ",")
}

func GetCpuRangeForNumaNode(nodeID int) (string, error) {
	if nodeID < 0 {
		return "", fmt.Errorf("NUMA node ID cannot be negative: %d", nodeID)
	}

	cpuListPath := filepath.Join("/sys/devices/system/node", fmt.Sprintf("node%d", nodeID), "cpulist")

	content, err := os.ReadFile(cpuListPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("NUMA node %d or its cpulist file not found at %s. Is NUMA enabled and node %d valid?", nodeID, cpuListPath, nodeID)
		}
		return "", fmt.Errorf("failed to read %s: %w", cpuListPath, err)
	}

	return strings.TrimSpace(string(content)), nil
}

func MergeCpuRanges(cpuRanges []string) (string, error) {
	var allCPUs []int
	for _, r := range cpuRanges {
		cpus, err := parseCpuRangeToSlice(r)
		if err != nil {
			return "", fmt.Errorf("failed to parse CPU range '%s': %w", r, err)
		}
		allCPUs = append(allCPUs, cpus...)
	}

	uniqueCPUsMap := make(map[int]bool)
	for _, cpu := range allCPUs {
		uniqueCPUsMap[cpu] = true
	}

	var uniqueSortedCPUs []int
	for cpu := range uniqueCPUsMap {
		uniqueSortedCPUs = append(uniqueSortedCPUs, cpu)
	}
	sort.Ints(uniqueSortedCPUs)

	return formatCpuSliceToRange(uniqueSortedCPUs), nil
}

func printDcusTopoInfo(dcuList []int, headerName string) {
	numaAffinityList := make([]int, 0, len(dcuList))
	numaNodeList := make([]int, 0, len(dcuList))
	for _, dcuIndex := range dcuList {
		numaInfoLst, err := dcgm.ShowNumaTopology([]int{dcuIndex})
		if err != nil {
			fmt.Printf("Error getting topo info: %v\n", err)
			return
		}
		numaAffinity := numaInfoLst[0].NumaAffinity
		numaNode := numaInfoLst[0].NumaNode
		numaAffinityList = append(numaAffinityList, numaAffinity)
		numaNodeList = append(numaNodeList, numaNode)
	}
	numaAffinityCompacted := slices.Compact(numaAffinityList)
	numaAffinityStr := make([]string, 0, len(numaAffinityCompacted))
	for _, numaAffinity := range numaAffinityCompacted {
		numaAffinityStr = append(numaAffinityStr, strconv.Itoa(numaAffinity))
	}
	numaNodeCompacted := slices.Compact(numaNodeList)
	var cpuAffinityList []string
	for _, numaNode := range numaNodeCompacted {
		cpuAffinityStr, err := GetCpuRangeForNumaNode(numaNode)
		if err != nil {
			fmt.Printf("Error getting topo info: %v\n", err)
			return

		}
		cpuAffinityList = append(cpuAffinityList, cpuAffinityStr)
	}
	cpuAffinityMerged, err := MergeCpuRanges(cpuAffinityList)
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	var numaOptimalStr string
	if len(numaNodeCompacted) == 1 {
		numaOptimalStr = "True"
	} else {
		numaOptimalStr = "False"
	}
	worstPathStr := "Connected via " + dcgm.LinkTypeXGMI
	if len(dcuList) == 1 {
		worstPathStr = "Unknown"
	} else {
		for _, dcuIndex := range dcuList {
			linkTypeList, err := dcgm.GetTopoLinkType(dcuIndex, dcuList)
			if err != nil {
				fmt.Printf("Error getting topo info: %v\n", err)
				return
			}
			for _, linkType := range linkTypeList {
				if linkType == dcgm.LinkTypePCIE {
					worstPathStr = "Connected via " + dcgm.LinkTypePCIE
				}
				break
			}
		}
	}
	fmt.Printf("+-------------------+------------------------------------------------------------------------------+\n"+
		"| Topology Information                                                                             |\n"+
		"| %-97s|\n+===================+==============================================================================+\n", headerName)
	fmt.Printf("| Numa Affinity     | %-77s|\n", strings.Join(numaAffinityStr, ","))
	fmt.Printf("| CPU Core Affinity | %-77s|\n", cpuAffinityMerged)
	fmt.Printf("| NUMA Optimal      | %-77s|\n", numaOptimalStr)
	fmt.Printf("| Worst Path        | %-77s|\n", worstPathStr)
	fmt.Println("+-------------------+------------------------------------------------------------------------------+")
}

func handleByGroup() {
	if dcuIndex != "" {
		fmt.Println("Error: Both GPU and group ID are given.")
		return
	}
	groupIdInt, err := strconv.Atoi(groupId)
	if err != nil {
		fmt.Println("Error: Invalid Group ID given.")
		return
	}
	dcuInGroup, groupName, err := dcgm.GetDcuListFromGroup(groupIdInt)
	if err != nil {
		fmt.Printf("Error getting group info: %v\n", err)
		return
	}
	if len(dcuInGroup) == 0 {
		fmt.Printf("Failed to query group: no entity found for group %v\n", groupId)
		return
	}
	printDcusTopoInfo(dcuInGroup, groupName)
}

func handleByIndex() {
	if groupId != "" {
		fmt.Println("Error: Both GPU and group ID are given.")
		return
	}
	numDcus, err := dcgm.NumMonitorDevices()
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	dcuList := make([]int, numDcus)
	dcuIndexInt, err := strconv.Atoi(dcuIndex)
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	for i := 0; i < numDcus; i++ {
		if i != dcuIndexInt {
			dcuList[i] = i
		}
	}
	linkTypeList, err := dcgm.GetTopoLinkType(dcuIndexInt, dcuList)
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	numaInfoList, err := dcgm.ShowNumaTopology([]int{dcuIndexInt})
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	numaNode := numaInfoList[0].NumaNode
	numaAffinity := numaInfoList[0].NumaAffinity
	cpuAffinity, err := GetCpuRangeForNumaNode(numaNode)
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	fmt.Printf("+-------------------+------------------------------------------------------------------------------+\n"+
		"| Topology Information                                                                             |\n"+
		"| DCU ID: %-89d|\n"+
		"+===================+==============================================================================+\n", dcuIndexInt)
	fmt.Printf("| Numa Node         | %-77d|\n", numaNode)
	fmt.Printf("| Numa Affinity     | %-77d|\n", numaAffinity)
	fmt.Printf("| CPU Core Affinity | %-77s|\n", cpuAffinity)
	for i := 0; i < len(linkTypeList); i++ {
		if i != dcuIndexInt {
			fmt.Printf("| To DCU %-11d| Connected via %-63s|\n", i, linkTypeList[i])
		}
	}
	fmt.Println("+-------------------+------------------------------------------------------------------------------+")

}

func handleAllDcu() {
	numDcus, err := dcgm.NumMonitorDevices()
	if err != nil {
		fmt.Printf("Error getting topo info: %v\n", err)
		return
	}
	dcuList := make([]int, numDcus)
	for i := 0; i < numDcus; i++ {
		dcuList[i] = i
	}
	printDcusTopoInfo(dcuList, "DCGM_ALL_SUPPORTED_DCUS")
}
