/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package cli

import (
	"fmt"
	"github.com/HYGON-AI/dcu-dcgm/v2/pkg/dcgm"
	"github.com/spf13/cobra"
	"slices"
	"strconv"
	"strings"
)

const NOT_APPLICABLE = "****"

var (
	infoFlags            string
	verboseFlag          bool
	computeHierarchyFlag bool
	dcuIndex             string
)

var discoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Used to discover and identify DCUs and their attributes.",
	Long:  `discovery -- Used to discover and identify DCUs and their attributes.`,
	Example: `  dcgmi discovery -l 
  dcgmi discovery -i <flags> --dcuid <dcuId> 
  dcgmi discovery -i <flags> -g <groupId> -v
  dcgmi discovery -c`,
	Run: func(cmd *cobra.Command, args []string) {
		// Main dispatcher logic
		switch {
		case listFlag:
			handleDiscoveryList()
		case computeHierarchyFlag:
			handleComputeHierarchy()
		case infoFlags != "":
			handleInfoFlagsOperations()
		default:
			cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(discoveryCmd)

	discoveryCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List all the DCUs discovery on the host.")
	discoveryCmd.Flags().StringVarP(&infoFlags, "info", "i", "", "Specify which information to return.\n"+
		" a - device info\n p - power limits\n t - thermal limits\n c - clocks")
	discoveryCmd.Flags().StringVar(&dcuIndex, "dcuid", "", "The DCU ID to query.")
	discoveryCmd.Flags().StringVarP(&groupId, "group", "g", "", "The group ID to query.")
	discoveryCmd.Flags().BoolVarP(&computeHierarchyFlag, "compute-hierarchy", "c", false, "List all of the gpu instances and compute instances.")
	discoveryCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Display information per DCU.")
}

// Handlers for subcommands
func handleDiscoveryList() {
	if computeHierarchyFlag {
		fmt.Printf("PARSE ERROR: Argument: -l AND -c provided.\n             Only one is allowed.\n")
		return
	}

	numDcus, err := dcgm.NumMonitorDevices()
	if err != nil {
		fmt.Printf("Error getting device count: %v\n", err)
		return
	}
	suffix := ""
	if numDcus != 1 {
		suffix = "s"
	}
	fmt.Printf("%d DCU%s found.\n", numDcus, suffix)
	fmt.Println("+--------+----------------------------------------------------------------------+")
	fmt.Println("| DCU ID | Device Information                                                   |")
	fmt.Println("+--------+----------------------------------------------------------------------+")

	for i := 0; i < numDcus; i++ {
		dcuName, err := dcgm.DevName(i)
		if err != nil {
			dcuName = "N/A"
		}
		dcuTypeName, _, err := dcgm.DevTypeName(i)
		if err != nil {
			dcuTypeName = "N/A"
		}
		nameStr := dcuName + " - " + dcuTypeName
		dcuUniqueId, err := dcgm.GetDeviceUniqueId(i)
		if err != nil {
			dcuUniqueId = "N/A"
		}
		pciId, err := dcgm.PicBusInfo(i)
		if err != nil {
			pciId = "N/A"
		}
		fmt.Printf("| %-7d| Name: %-63s|\n", i, nameStr)
		fmt.Printf("|        | PCI Bus ID: %-57s|\n", pciId)
		fmt.Printf("|        | Device Unique ID: %-51s|\n", dcuUniqueId)
		fmt.Println("+--------+----------------------------------------------------------------------+")
	}
}

func handleComputeHierarchy() {
	if groupId != "" || dcuIndex != "" {
		fmt.Printf("For now, hierarchy must be used by itself.\n")
		return
	}

	if infoFlags != "" {
		fmt.Printf("PARSE ERROR: Argument: -i AND -c provided.\n             Only one is allowed.\n")
		return
	}

	fmt.Printf("+-------------------+--------------------------------------------------------------------+\n" +
		"| Instance Hierarchy                                                                     |\n" +
		"+===================+====================================================================+\n")
	migMode, _, err := dcgm.SystemMigMode()
	if err != nil || migMode != 1 {
		fmt.Println("+-------------------+--------------------------------------------------------------------+")
		return
	}
	migDevicesInfos, err := dcgm.MigInfos()
	if err != nil {
		fmt.Println("+-------------------+--------------------------------------------------------------------+")
		return
	}
	numDcus, err := dcgm.NumMonitorDevices()
	if err != nil {
		fmt.Println("+-------------------+--------------------------------------------------------------------+")
		return
	}
	migInfoList := make([][]dcgm.MigInfo, numDcus)
	for _, migInfo := range migDevicesInfos {
		migInfoList[migInfo.DvInd] = append(migInfoList[migInfo.DvInd], migInfo)
	}
	for i := 0; i < numDcus; i++ {
		if len(migInfoList[i]) > 0 {
			dcuUniqueId, err := dcgm.GetDeviceUniqueId(i)
			if err != nil {
				dcuUniqueId = "N/A"
			}
			fmt.Printf("| DCU %-14d| DCU %-63s|\n", i, dcuUniqueId)
			for _, migInfo := range migInfoList[i] {
				giIndexStr := fmt.Sprintf("%d/%d", i, migInfo.GpuInstanceId)
				ciIndexStr := fmt.Sprintf("%d/%d/%d", i, migInfo.GpuInstanceId, migInfo.ComputeInstanceId)
				ciStr := fmt.Sprintf("Compute Instance (%s) created by Profile %d", migInfo.Name, migInfo.CiProfileId)
				fmt.Printf("| -> I %-13s| GPU Instance created by Profile %-35d|\n", giIndexStr, migInfo.GiProfileId)
				fmt.Printf("|    -> CI %-9s| %-66s |\n", ciIndexStr, ciStr)
			}
			fmt.Println("+-------------------+--------------------------------------------------------------------+")
		}
	}
}

func queryDcuInfo(dcuIndex int, infoFlags string) error {
	fmt.Printf("+--------------------------+-------------------------------------------------+\n"+
		"| DCU ID: %-17d| Device Information                              |\n"+
		"+==========================+=================================================+\n", dcuIndex)
	if strings.Contains(infoFlags, "a") {
		nameStr, pciId, dcuUniqueId, dcuSerialNumber, vbiosVersion := getIdentifiers(dcuIndex)
		printIdentifiers(nameStr, pciId, dcuUniqueId, dcuSerialNumber, vbiosVersion)
	}
	if strings.Contains(infoFlags, "p") {
		powerAveStr, powerMaxStr, powerMinStr := getPowerLimits(dcuIndex)
		printPowerLimits(powerAveStr, powerMaxStr, powerMinStr)
	}
	if strings.Contains(infoFlags, "t") {
		tempCurrentStr, tempShutdownStr, tempCriticalStr, tempSlowdownStr := getThermals(dcuIndex)
		printThermals(tempCurrentStr, tempShutdownStr, tempCriticalStr, tempSlowdownStr)
	}
	if strings.Contains(infoFlags, "c") {
		mclkStrList, sclkStrList := getClocks(dcuIndex)
		printClocks(mclkStrList, sclkStrList)
	}
	return nil
}

func queryGroupInfo(groupId int, infoFlags string, verboseFlag bool) error {
	groupInfo, err := dcgm.GetGroupInfo(groupId)
	if err != nil {
		return err
	}
	entityList := groupInfo.EntityList
	if len(entityList) == 0 {
		return fmt.Errorf("no entity found for group %d", groupId)
	}
	var dcuInGroup []int
	for _, entity := range entityList {
		if entity.EntityGroupId == dcgm.FE_DCU {
			dcuInGroup = append(dcuInGroup, entity.EntityId)
		}
	}
	if verboseFlag {
		for _, entityId := range dcuInGroup {
			err = queryDcuInfo(entityId, infoFlags)
			if err != nil {
				return err
			}
		}
	} else {
		err = queryNonVerboseGroupInfo(infoFlags, dcuInGroup)
		if err != nil {
			return err
		}
	}
	return nil
}

func getIdentifiers(dcuIndex int) (nameStr, pciId, dcuUniqueId, dcuSerialNumber, vbiosVersion string) {
	dcuName, err := dcgm.DevName(dcuIndex)
	if err != nil {
		dcuName = "N/A"
	}
	dcuTypeName, _, err := dcgm.DevTypeName(dcuIndex)
	if err != nil {
		dcuTypeName = "N/A"
	}
	nameStr = dcuName + " - " + dcuTypeName
	pciId, err = dcgm.PicBusInfo(dcuIndex)
	if err != nil {
		pciId = "N/A"
	}
	dcuUniqueId, err = dcgm.GetDeviceUniqueId(dcuIndex)
	if err != nil {
		dcuUniqueId = "N/A"
	}
	dcuSerialNumber, err = dcgm.GetDeviceId(dcuIndex)
	if err != nil {
		dcuSerialNumber = "N/A"
	}
	vbiosVersion, err = dcgm.VbiosVersion(dcuIndex)
	if err != nil {
		vbiosVersion = "N/A"
	}
	return
}

func printIdentifiers(nameStr, pciId, dcuUniqueId, dcuSerialNumber, vbiosVersion string) {
	fmt.Printf("| Device Name              | %-48s|\n", nameStr)
	fmt.Printf("| PCI Bus ID               | %-48s|\n", pciId)
	fmt.Printf("| Unique ID                | %-48s|\n", dcuUniqueId)
	fmt.Printf("| Serial Number            | %-48s|\n", dcuSerialNumber)
	fmt.Printf("| VBIOS                    | %-48s|\n", vbiosVersion)
	fmt.Println("+--------------------------+-------------------------------------------------+")
}

func getPowerLimits(dcuIndex int) (powerAveStr, powerMaxStr, powerMinStr string) {
	powerAve, err := dcgm.Power(dcuIndex)
	if err != nil {
		powerAveStr = "N/A"
	} else {
		powerAveStr = strconv.Itoa(int(powerAve))
	}

	powerMax, powerMin, err := dcgm.DevPowerCapRange(dcuIndex)
	if err != nil {
		powerMinStr = "N/A"
		powerMaxStr = "N/A"
	} else {
		powerMax = (powerMax / 1000000)
		powerMin = (powerMin / 1000000)
		powerMaxStr = strconv.Itoa(int(powerMax))
		powerMinStr = strconv.Itoa(int(powerMin))
	}
	return
}

func printPowerLimits(powerAveStr, powerMaxStr, powerMinStr string) {
	fmt.Printf("| Power Ave Value (W)      | %-48s|\n", powerAveStr)
	fmt.Printf("| Power Max Value (W)      | %-48s|\n", powerMaxStr)
	fmt.Printf("| Power Min Value (W)      | %-48s|\n", powerMinStr)
	fmt.Println("+--------------------------+-------------------------------------------------+")
}

func getThermals(dcuIndex int) (tempCurrentStr, tempShutdownStr, tempCriticalStr, tempSlowdownStr string) {
	tempCurrent, err := dcgm.GetTempByMetric(dcuIndex, dcgm.RSMI_TEMP_CURRENT)
	if err != nil {
		tempCurrentStr = "N/A"
	} else {
		tempCurrentStr = strconv.FormatFloat(tempCurrent, 'f', -1, 64)
	}

	tempSlowdown, err := dcgm.GetTempByMetric(dcuIndex, dcgm.RSMI_TEMP_MAX)
	if err != nil {
		tempSlowdownStr = "N/A"
	} else {
		tempSlowdownStr = strconv.FormatFloat(tempSlowdown, 'f', -1, 64)
	}

	tempCritical, err := dcgm.GetTempByMetric(dcuIndex, dcgm.RSMI_TEMP_CRITICAL)
	if err != nil {
		tempCriticalStr = "N/A"
	} else {
		tempCriticalStr = strconv.FormatFloat(tempCritical, 'f', -1, 64)
	}

	tempShutdown, err := dcgm.GetTempByMetric(dcuIndex, dcgm.RSMI_TEMP_EMERGENCY)
	if err != nil {
		tempShutdownStr = "N/A"
	} else {
		tempShutdownStr = strconv.FormatFloat(tempShutdown, 'f', -1, 64)
	}
	return
}

func printThermals(tempCurrentStr, tempShutdownStr, tempCriticalStr, tempSlowdownStr string) {
	fmt.Printf("| Current Temperature (C)  | %-48s|\n", tempCurrentStr)
	fmt.Printf("| ShutDown Temperature (C) | %-48s|\n", tempShutdownStr)
	fmt.Printf("| Critical Temperature (C) | %-48s|\n", tempCriticalStr)
	fmt.Printf("| Slowdown Temperature (C) | %-48s|\n", tempSlowdownStr)
	fmt.Println("+--------------------------+-------------------------------------------------+")
}

func getClocks(dcuIndex int) (mclkStrList, sclkStrList []string) {
	mclkList, mclkCurrent, err := dcgm.GetClocksByType(dcuIndex, dcgm.RSMI_CLK_TYPE_MEM)

	if err != nil {
		mclkStrList = append(mclkStrList, "N/A")
	} else {
		for index, mclk := range mclkList {
			mclkStr := strconv.FormatUint(mclk, 10)
			if int(mclkCurrent) == index {
				mclkStr = mclkStr + " *"
			}
			mclkStrList = append(mclkStrList, mclkStr)
		}
	}
	sclkList, sclkCurrent, err := dcgm.GetClocksByType(dcuIndex, dcgm.RSMI_CLK_TYPE_SYS)

	if err != nil {
		sclkStrList = append(sclkStrList, "N/A")
	} else {
		for index, sclk := range sclkList {
			sclkStr := strconv.FormatUint(sclk, 10)
			if int(sclkCurrent) == index {
				sclkStr = sclkStr + " *"
			}
			sclkStrList = append(sclkStrList, sclkStr)
		}
	}
	return
}

func printClocks(mclkStrList, sclkStrList []string) {
	fmt.Println("| Supported Clocks (MHz)   | MCLK:                                           |")
	for _, mclk := range mclkStrList {
		fmt.Printf("|                          | %-48s|\n", mclk)
	}
	fmt.Println("|                          |                                                 |")
	fmt.Println("|                          | SCLK:                                           |")
	for _, sclk := range sclkStrList {
		fmt.Printf("|                          | %-48s|\n", sclk)
	}
	fmt.Println("+--------------------------+-------------------------------------------------+")
}

func queryNonVerboseGroupInfo(infoFlags string, dcuInGroup []int) error {
	fmt.Println("+--------------------------+-------------------------------------------------+")
	if len(dcuInGroup) == 1 {
		fmt.Println("| Group of 1 DCU           | Device Information                              |")

	} else {
		fmt.Printf("| Group of %d DCUs          | Device Information                              |\n", len(dcuInGroup))
	}
	fmt.Println("+==========================+=================================================+")
	if strings.Contains(infoFlags, "a") {
		tmpNameStr, tmpPciId, tmpDcuUniqueId, tmpDcuSerialNumber, tmpVbiosVersion := getIdentifiers(dcuInGroup[0])

		for i := 1; i < len(dcuInGroup); i++ {
			nameStr, pciId, dcuUniqueId, dcuSerialNumber, vbiosVersion := getIdentifiers(dcuInGroup[i])
			if tmpNameStr != NOT_APPLICABLE && tmpNameStr != nameStr {
				tmpNameStr = NOT_APPLICABLE
			}
			if tmpPciId != NOT_APPLICABLE && tmpPciId != pciId {
				tmpPciId = NOT_APPLICABLE
			}
			if tmpDcuUniqueId != NOT_APPLICABLE && tmpDcuUniqueId != dcuUniqueId {
				tmpDcuUniqueId = NOT_APPLICABLE
			}
			if tmpDcuSerialNumber != NOT_APPLICABLE && tmpDcuSerialNumber != dcuSerialNumber {
				tmpDcuSerialNumber = NOT_APPLICABLE
			}
			if tmpVbiosVersion != NOT_APPLICABLE && tmpVbiosVersion != vbiosVersion {
				tmpVbiosVersion = NOT_APPLICABLE
			}
		}
		printIdentifiers(tmpNameStr, tmpPciId, tmpDcuUniqueId, tmpDcuSerialNumber, tmpVbiosVersion)
	}
	if strings.Contains(infoFlags, "p") {
		tmpPowerAve, tmpPowerMax, tmpPowerMin := getPowerLimits(dcuInGroup[0])
		for i := 1; i < len(dcuInGroup); i++ {
			powerAve, powerMax, powerMin := getPowerLimits(dcuInGroup[i])
			if tmpPowerAve != NOT_APPLICABLE && tmpPowerAve != powerAve {
				tmpPowerAve = NOT_APPLICABLE
			}
			if tmpPowerMax != NOT_APPLICABLE && tmpPowerMax != powerMax {
				tmpPowerMax = NOT_APPLICABLE
			}
			if tmpPowerMin != NOT_APPLICABLE && tmpPowerMin != powerMin {
				tmpPowerMin = NOT_APPLICABLE
			}
		}
		printPowerLimits(tmpPowerAve, tmpPowerMax, tmpPowerMin)
	}
	if strings.Contains(infoFlags, "t") {
		tmpTempCurrent, tmpTempShutdown, tmpTempCritical, tmpTempSlowdown := getThermals(dcuInGroup[0])
		for i := 1; i < len(dcuInGroup); i++ {
			tempCurrent, tempShutdown, tempCritical, tempSlowdown := getThermals(dcuInGroup[i])
			if tmpTempCurrent != NOT_APPLICABLE && tmpTempCurrent != tempCurrent {
				tmpTempCurrent = NOT_APPLICABLE
			}
			if tmpTempShutdown != NOT_APPLICABLE && tmpTempShutdown != tempShutdown {
				tmpTempShutdown = NOT_APPLICABLE
			}
			if tmpTempCritical != NOT_APPLICABLE && tmpTempCritical != tempCritical {
				tmpTempCritical = NOT_APPLICABLE
			}
			if tmpTempSlowdown != NOT_APPLICABLE && tmpTempSlowdown != tempSlowdown {
				tmpTempSlowdown = NOT_APPLICABLE
			}
		}
		printThermals(tmpTempCurrent, tmpTempShutdown, tmpTempCritical, tmpTempSlowdown)
	}
	if strings.Contains(infoFlags, "c") {
		tmpMclkList, tmpSclkList := getClocks(dcuInGroup[0])
		for i := 1; i < len(dcuInGroup); i++ {
			mclkList, sclkList := getClocks(dcuInGroup[i])
			if tmpMclkList[0] != NOT_APPLICABLE && !slices.Equal(tmpMclkList, mclkList) {
				tmpMclkList = []string{NOT_APPLICABLE}
			}
			if tmpSclkList[0] != NOT_APPLICABLE && !slices.Equal(tmpSclkList, sclkList) {
				tmpSclkList = []string{NOT_APPLICABLE}
			}
		}
		printClocks(tmpMclkList, tmpSclkList)
	}
	fmt.Println("**** Non-homogenous settings across group. Use with –v flag to see details.")
	return nil
}

func queryAllDcusInfo(infoFlags string) error {
	numDcus, err := dcgm.NumMonitorDevices()
	if err != nil {
		return err
	}
	for i := 0; i < numDcus; i++ {
		err = queryDcuInfo(i, infoFlags)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleInfoFlagsOperations() {
	for _, c := range infoFlags {
		switch c {
		case 'a', 't', 'p', 'c':
		default:
			fmt.Printf("Invalid input '%c'. Please include only valid tags.\n", c)
			return
		}
	}

	if groupId != "" && dcuIndex != "" {
		fmt.Printf("Both DCU and Group specified at command line.\n")
		return
	}

	if groupId != "" {
		groupId, err := strconv.Atoi(groupId)
		if err != nil {
			fmt.Printf("Invalid Group ID specified.\n")
			return
		}
		err = queryGroupInfo(groupId, infoFlags, verboseFlag)
		if err != nil {
			fmt.Printf("Failed to query group: %v\n", err)
			return
		}
	} else if dcuIndex != "" {
		dcuIndex, err := strconv.Atoi(dcuIndex)
		if err != nil {
			fmt.Printf("Invalid DCU index specified.\n")
			return
		}
		err = queryDcuInfo(dcuIndex, infoFlags)
		if err != nil {
			fmt.Printf("Failed to query DCU: %v\n", err)
			return
		}
	} else if verboseFlag {
		err := queryAllDcusInfo(infoFlags)
		if err != nil {
			fmt.Printf("Failed to query all DCUs: %v\n", err)
			return
		}
	}

}
