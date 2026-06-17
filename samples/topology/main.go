/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Hygon Information Technology Co., Ltd.
 */
package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"

	"github.com/HYGON-AI/dcu-dcgm/pkg/dcgm"
)

func main() {
	flag.Parse()
	defer glog.Flush()
	glog.Info("go-dcgm start ...")
	//初始化dcgm服务
	dcgm.Init()
	defer dcgm.ShutDown()

	//硬件拓扑信息(支持json打印信息)
	//dcgm.ShowWeightTopology([]int{0, 1, 2}, true)
	//dcgm.ShowWeightTopology([]int{0, 1, 2}, false)
	//基于跳数显示硬件拓扑信息(支持json打印信息)
	//dcgm.ShowHopsTopology([]int{0, 1, 2}, false)
	//dcgm.ShowHopsTopology([]int{0, 1, 2}, true)
	//基于链接类型的硬件拓扑信息(支持json打印信息)
	//dcgm.ShowTypeTopology([]int{0, 1, 2}, true)
	//dcgm.ShowTypeTopology([]int{0, 1, 2}, false)
	//numa节点HW拓扑信息
	//dcgm.ShowNumaTopology([]int{0, 1})
	//显示硬件拓扑信息,包括权重、跳数、链接类型以及NUMA节点信息
	//dcgm.ShowHwTopology([]int{0, 1, 2})

	// 获取 DCU 数量
	devices, err := dcgm.NumMonitorDevices()
	if err != nil {
		glog.Errorf("NumMonitorDevices failed: %v", err)
		return
	}

	// matrix[src][dst]
	matrix := make([][]string, devices)
	for i := 0; i < devices; i++ {
		matrix[i] = make([]string, devices)
	}

	// 构建拓扑矩阵
	for src := 0; src < devices; src++ {
		for dst := 0; dst < devices; dst++ {

			// 自己到自己
			if src == dst {
				matrix[src][dst] = "None"
				continue
			}

			linkTypes, err := dcgm.GetTopoLinkType(src, []int{dst})
			if err != nil || len(linkTypes) == 0 {
				matrix[src][dst] = "ERR"
				continue
			}

			linkType := linkTypes[0]

			// 只有 XGMI 才判断 hyswitch
			if linkType == "XGMI" {
				isHylink, err := dcgm.TopoIsHylink(src, dst)
				if err != nil {
					matrix[src][dst] = "XGMI"
				} else if isHylink {
					matrix[src][dst] = "hyswitch"
				} else {
					matrix[src][dst] = "XGMI"
				}
			} else {
				// PCIE / Unknown
				matrix[src][dst] = linkType
			}
		}
	}

	// 打印拓扑表格
	//printTopoMatrix(matrix)

	// 遍历所有 DCU
	for dvInd := 0; dvInd < devices; dvInd++ {

		// 获取当前 DCU 的 XHCL 链路状态
		states, err := dcgm.XhclLinkStates(dvInd)
		if err != nil {
			glog.Errorf("获取 DCU %d XHCL 链路状态失败: %v", dvInd, err)
			continue
		}

		// 打印每条链路状态
		for _, s := range states {
			status := "DOWN"
			if s.Up {
				status = "UP"
			}

			glog.V(5).Infof(
				"DCU %d XHCL 链路 %d/%d 状态: %s (原始 state=%d)",
				dvInd,
				s.LinkID+1, // 用户显示从 1 开始
				len(states),
				status,
				s.State,
			)
		}

		// 每张卡一个 summary
		fmt.Printf(
			"DCU %d XHCL 链路总数: %d\n",
			dvInd,
			len(states),
		)
	}

	// 可选：在终端显示一条 summary
	//fmt.Printf("GPU %d XHCL 链路总数: %d\n", dvInd, len(states))

	for dvInd := 0; dvInd < devices; dvInd++ {
		glog.Infof("===== DCU %d XHCL Remote Devices =====", dvInd)

		remotes, err := dcgm.DumpXhclRemoteBdfids(dvInd)
		if err != nil {
			glog.Errorf(
				"DumpXhclRemoteBdfids failed for DCU %d: %v",
				dvInd,
				err,
			)
			continue
		}

		for _, r := range remotes {
			fmt.Printf(
				"DCU %d link %d -> remote BDFID: 0x%x\n",
				dvInd,
				r.LinkID,
				r.BdfID,
			)
		}
	}
	/*****************************************************/
	// 构建邻接表
	neighborTable := make([][]string, devices)
	for i := 0; i < devices; i++ {
		neighborTable[i] = make([]string, devices)
		for j := 0; j < devices; j++ {
			neighborTable[i][j] = "-" // 默认
		}
	}

	// 填充远端 BDFID
	for src := 0; src < devices; src++ {
		links, err := dcgm.DumpXhclRemoteBdfids(src)
		if err != nil {
			glog.Warningf("DumpXhclRemoteBdfids failed for DCU %d: %v", src, err)
			continue
		}

		for _, link := range links {
			remoteBdf := link.BdfID
			bus := (remoteBdf >> 8) & 0xff
			device := (remoteBdf >> 3) & 0x1f
			function := remoteBdf & 0x7

			neighborTable[src][link.LinkID] = fmt.Sprintf("%02x:%02x.%x", bus, device, function)
		}
	}

	// 打印列表（每个 DCU 下列出它每条 XHCL Link 对应的远端 BDFID）
	//fmt.Println("\nDCU ↔ DCU XHCL Neighbor List:\n")

	for i := 0; i < devices; i++ {
		fmt.Printf("Device DCU[%d]:\n", i)
		for linkID, bdf := range neighborTable[i] {
			if bdf == "-" {
				continue
			}
			fmt.Printf("  Link %d: Remote device %s\n", linkID, bdf)
		}
		fmt.Println()
	}

	return

}
func printTopoMatrix(matrix [][]string) {
	devices := len(matrix)

	// 表头
	fmt.Printf("%-8s", "")
	for i := 0; i < devices; i++ {
		fmt.Printf("%-10s", fmt.Sprintf("DCU[%d]", i))
	}
	fmt.Println()

	// 表内容
	for i := 0; i < devices; i++ {
		fmt.Printf("%-8s", fmt.Sprintf("DCU[%d]", i))
		for j := 0; j < devices; j++ {
			fmt.Printf("%-10s", matrix[i][j])
		}
		fmt.Println()
	}
}
