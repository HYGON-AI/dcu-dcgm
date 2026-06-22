/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package cli

import (
	"fmt"
	"github.com/HYGON-AI/dcu-dcgm/v2/pkg/dcgm"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "View available profiling metrics for DCUs",
	Long:  `profile -- View available profiling metrics for DCUs`,
	Example: `  dcgmi profile -l 
  dcgmi profile -l -i <entityId>
  dcgmi profile -l -g <groupId>`,
	Run: func(cmd *cobra.Command, args []string) {
		if !listFlag {
			fmt.Println("PARSE ERROR: Required argument missing: list")
			return
		} else if entityId != "" && groupId != "" {
			fmt.Println("Error: Both entity and group IDs specified. Please use only one at a time.")
			return
		} else if entityId != "" {
			handleProfileWithEntity()
			return
		} else if groupId != "" {
			handleProfileWithGroup()
			return
		} else {
			handleProfileList()
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List available profiling metrics")
	profileCmd.Flags().StringVarP(&entityId, "entityId", "i", "", "Comma-seperated list of entity IDs to query.\n"+
		"Default is supported DCUs on the system. Run\ndcgmi discovery -l to check list of DCUs available")
	profileCmd.Flags().StringVarP(&groupId, "group", "g", "", "The group of DCUs to query on the host.")
}

func setDcuIdFromEntityList(entityId string) (dcuIndex int, err error) {
	entityList, err := parseEntityList(entityId)
	if err != nil {
		return 0, fmt.Errorf("Invalid --entityid input: %v\n", err)
	}
	for _, entity := range entityList {
		if entity.EntityGroupId == dcgm.FE_DCU {
			mDcuId := entity.EntityId
			return mDcuId, nil
		}
	}
	return 0, fmt.Errorf("Error: No DCUs found in the provided entity list.")
}

func printProfilingMetrics(dcuIndex int) {
	metricGroups, err := dcgm.GetSupportedMetricGroups(dcuIndex)
	if err != nil {
		fmt.Printf("Error getting supported metric groups: %v\n", err)
		return
	}
	fmt.Printf("+----------------+----------+------------------------------------------------------+\n" +
		"| Group.Subgroup | Field ID | Field Tag                                            |\n" +
		"+----------------+----------+------------------------------------------------------+\n")
	for _, metricGroup := range metricGroups {
		groupStr := fmt.Sprintf("%s.%d", string(rune('A'-1+metricGroup.Major)), metricGroup.Minor)
		for _, fieldId := range metricGroup.FieldIds {
			fieldName := dcgm.FieldIdToName[fieldId]
			fmt.Printf("| %-15s| %-9d| %-53s|\n", groupStr, fieldId, strings.ToLower(strings.TrimPrefix(fieldName, "DCU_")))
		}
	}
	fmt.Println("+----------------+----------+------------------------------------------------------+")
}

func handleProfileWithEntity() {
	dcuIndex, err := setDcuIdFromEntityList(entityId)
	if err != nil {
		fmt.Println(err)
		return
	}
	printProfilingMetrics(dcuIndex)
}

func handleProfileWithGroup() {
	groupId, err := strconv.Atoi(groupId)
	if err != nil {
		fmt.Printf("Invalid Group ID specified.\n")
		return
	}
	dcuInGroup, _, err := dcgm.GetDcuListFromGroup(groupId)
	if err != nil {
		fmt.Printf("Error getting group info: %v\n", err)
		return
	}
	if len(dcuInGroup) == 0 {
		fmt.Printf("Failed to query group: no DCU found for group %v\n", groupId)
		return
	}
	printProfilingMetrics(dcuInGroup[0])
}

func handleProfileList() {
	numDevices, err := dcgm.NumMonitorDevices()
	if err != nil {
		fmt.Printf("Failed to get DCUs: %v\n", err)
		return
	}
	if numDevices < 1 {
		fmt.Println("Error: found 0 DCUs")
		return
	}
	printProfilingMetrics(0)
}
