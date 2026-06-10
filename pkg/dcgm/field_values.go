/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package dcgm

import "fmt"

func getFieldValue(fieldEntityGroup Field_Entity_Group, entityId int, fieldId int) (value float64, err error) {
	switch fieldEntityGroup {
	case FE_DCU:
		return getDcuFieldValue(entityId, fieldId)
	case FE_VDCU:
		return getVdcuFieldValue(entityId, fieldId)
	default:
		err = fmt.Errorf("currently not supported field entity group %s", fieldEntityGroup.String())
		return
	}
}

func getDcuFieldValue(dcuIndex int, fieldId int) (value float64, err error) {
	switch fieldId {
	case DCU_TEMP:
		return Temperature(dcuIndex)
	case DCU_POWER_USAGE:
		power, err := Power(dcuIndex)
		return float64(power), err
	case DCU_POWER_CAP:
		powerCap, err := MaxPower(dcuIndex)
		return float64(powerCap), err
	case DCU_UTILIZATION_RATE:
		utilizationRate, err := DCUUse(dcuIndex)
		return float64(utilizationRate), err
	case DCU_USED_MEMORY_BYTES:
		return MemoryUsed(dcuIndex)
	case DCU_MEMORY_CAP_BYTES:
		return MemoryTotal(dcuIndex)
	case DCU_MEMORY_PERCENT:
		memoryPercent, err := MemoryPercent(dcuIndex)
		return float64(memoryPercent), err
	case DCU_PCIE_BW_MB, DCU_PCIE_RECEIVE_MB, DCU_PCIE_SENT_MB:
		pcieBandwidth, err := PcieBw(dcuIndex)
		if err != nil {
			return 0, err
		}
		switch fieldId {
		case DCU_PCIE_BW_MB:
			return pcieBandwidth.Sent + pcieBandwidth.Received, nil
		case DCU_PCIE_RECEIVE_MB:
			return pcieBandwidth.Received, nil
		case DCU_PCIE_SENT_MB:
			return pcieBandwidth.Sent, nil
		}
	case DCU_SCLK:
		return DCUClk(dcuIndex)
	case DCU_COMPUTE_UNIT_COUNT:
		deviceInfo, err := GetDeviceInfo(dcuIndex)
		return float64(deviceInfo.ComputeUnitCount), err
	case DCU_COMPUTE_UNIT_REMAINING_COUNT, DCU_MEMORY_REMAINING:
		cus, mem, err := DeviceRemainingInfo(dcuIndex)
		if err != nil {
			return 0, err
		}
		if fieldId == DCU_COMPUTE_UNIT_REMAINING_COUNT {
			return float64(cus), nil
		}
		return float64(mem), nil
	case DCU_DF_BW_READ, DCU_DF_BW_WRITE, DCU_DF_BW_READ_WRITE:
		bandwidth, err := DFBandwidth(dcuIndex, RSMI_DF_BW_TYPE_ALL)
		if err != nil {
			return 0, err
		}
		switch fieldId {
		case DCU_DF_BW_READ:
			return bandwidth.ReadBW, nil
		case DCU_DF_BW_WRITE:
			return bandwidth.WriteBW, nil
		case DCU_DF_BW_READ_WRITE:
			return bandwidth.ReadWriteBW, nil
		}
	case DCU_VDCU_COUNT:
		vDeviceCount, _, err := VDeviceByDvInd(dcuIndex)
		return float64(vDeviceCount), err
	default:
		err = fmt.Errorf("unknown field %v", fieldId)
		return 0, err
	}
	return 0, nil
}

func getVdcuFieldValue(vdcuIndex int, fieldId int) (value float64, err error) {
	vDcuInfo, err := dmiGetVDeviceInfo(vdcuIndex)
	if err != nil {
		return 0, err
	}
	switch fieldId {
	case VDCU_SCLK:
		return DCUClk(vDcuInfo.DvInd)
	case VDCU_TEMP:
		return Temperature(vDcuInfo.DvInd)
	case VDCU_UTILIZATION_RATE:
		return float64(vDcuInfo.VPercent), nil
	case VDCU_USED_MEMORY_BYTES:
		return float64(vDcuInfo.VMemoryUsed), nil
	default:
		err = fmt.Errorf("unknown field %v", fieldId)
		return 0, err
	}
}
