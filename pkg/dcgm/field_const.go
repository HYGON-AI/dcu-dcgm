/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package dcgm

const (
	DCU_TEMP int = iota + 101
	DCU_POWER_USAGE
	DCU_POWER_CAP
	DCU_UTILIZATION_RATE
	DCU_SCLK
	DCU_COMPUTE_UNIT_COUNT
	DCU_COMPUTE_UNIT_REMAINING_COUNT
	DCU_USED_MEMORY_BYTES
	DCU_MEMORY_CAP_BYTES
	DCU_MEMORY_REMAINING
	DCU_MEMORY_PERCENT
	DCU_PCIE_BW_MB
	DCU_PCIE_RECEIVE_MB
	DCU_PCIE_SENT_MB
	DCU_DF_BW_READ
	DCU_DF_BW_WRITE
	DCU_DF_BW_READ_WRITE
	DCU_VDCU_COUNT
)

const (
	VDCU_SCLK int = iota + 201
	VDCU_TEMP
	VDCU_UTILIZATION_RATE
	VDCU_USED_MEMORY_BYTES
)

var dcgmFields = map[string]int{
	"DCU_TEMP":                         DCU_TEMP,
	"DCU_POWER_USAGE":                  DCU_POWER_USAGE,
	"DCU_POWER_CAP":                    DCU_POWER_CAP,
	"DCU_UTILIZATION_RATE":             DCU_UTILIZATION_RATE,
	"DCU_SCLK":                         DCU_SCLK,
	"DCU_COMPUTE_UNIT_COUNT":           DCU_COMPUTE_UNIT_COUNT,
	"DCU_COMPUTE_UNIT_REMAINING_COUNT": DCU_COMPUTE_UNIT_REMAINING_COUNT,
	"DCU_USED_MEMORY_BYTES":            DCU_USED_MEMORY_BYTES,
	"DCU_MEMORY_CAP_BYTES":             DCU_MEMORY_CAP_BYTES,
	"DCU_MEMORY_REMAINING":             DCU_MEMORY_REMAINING,
	"DCU_MEMORY_PERCENT":               DCU_MEMORY_PERCENT,
	"DCU_PCIE_BW_MB":                   DCU_PCIE_BW_MB,
	"DCU_PCIE_RECEIVE_MB":              DCU_PCIE_RECEIVE_MB,
	"DCU_PCIE_SENT_MB":                 DCU_PCIE_SENT_MB,
	"DCU_DF_BW_READ":                   DCU_DF_BW_READ,
	"DCU_DF_BW_WRITE":                  DCU_DF_BW_WRITE,
	"DCU_DF_BW_READ_WRITE":             DCU_DF_BW_READ_WRITE,
	"DCU_VDCU_COUNT":                   DCU_VDCU_COUNT,
	"VDCU_SCLK":                        VDCU_SCLK,
	"VDCU_TEMP":                        VDCU_TEMP,
	"VDCU_UTILIZATION_RATE":            VDCU_UTILIZATION_RATE,
	"VDCU_USED_MEMORY_BYTES":           VDCU_USED_MEMORY_BYTES,
}

var FieldIdToName map[int]string
var profilingMetrics []MetricGroup

var unsupportedFieldsByName = map[string][]int{
	"K100AI": {},
	"K100":   {},
	"Z100":   {},
	"BW":     {},
}

func init() {
	FieldIdToName = make(map[int]string, len(dcgmFields))
	for name, id := range dcgmFields {
		FieldIdToName[id] = name
	}
	profilingMetrics = []MetricGroup{
		MetricGroup{1, 1, []int{DCU_UTILIZATION_RATE, DCU_SCLK}},
		{2, 1, []int{DCU_MEMORY_PERCENT}},
		{3, 1, []int{DCU_PCIE_BW_MB, DCU_PCIE_SENT_MB, DCU_PCIE_RECEIVE_MB}},
	}
}

func getFieldId(fieldName string) (int, bool) {
	id, ok := dcgmFields[fieldName]
	return id, ok
}
