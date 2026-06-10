/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package dcgm

/*
#cgo CFLAGS: -Wall -I./include
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include "driver_caps.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/golang/glog"
)

// driverCaps 记录当前已加载驱动 .so 中可选 API 是否可用。
type driverCaps struct {
	HasXhclBandwidth bool
	HasUmcBandwidth  bool
}

var (
	driverCapability driverCaps
	capsOnce         sync.Once
)

var (
	errXhclBandwidthUnsupported = fmt.Errorf("driver does not support xhcl bandwidth (rsmi_dev_xhcl_bandwidth_get)")
	errUmcBandwidthUnsupported  = fmt.Errorf("driver does not support umc bandwidth (rsmi_dev_umc_bandwidth_get)")
)

// probeDriverCaps 在 Init 成功后探测可选 RSMI 符号，仅执行一次。
func probeDriverCaps() {
	capsOnce.Do(func() {
		driverCapability.HasXhclBandwidth = rsmiSymbolExists("rsmi_dev_xhcl_bandwidth_get")
		driverCapability.HasUmcBandwidth = rsmiSymbolExists("rsmi_dev_umc_bandwidth_get")
		glog.Infof("driver caps: xhcl_bandwidth=%v umc_bandwidth=%v",
			driverCapability.HasXhclBandwidth, driverCapability.HasUmcBandwidth)
	})
}

func rsmiSymbolExists(name string) bool {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.dcgm_lookup_symbol(cname) != nil
}

func ensureXhclBandwidth() error {
	probeDriverCaps()
	if !driverCapability.HasXhclBandwidth {
		return errXhclBandwidthUnsupported
	}
	return nil
}

func ensureUmcBandwidth() error {
	probeDriverCaps()
	if !driverCapability.HasUmcBandwidth {
		return errUmcBandwidthUnsupported
	}
	return nil
}
