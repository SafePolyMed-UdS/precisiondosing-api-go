package srvstats

import (
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

type CPUStats struct {
	UsagePerCore []float64 `json:"usage_per_core"` // CPU usage percentage
	Usage        []float64 `json:"usage"`          // CPU usage percentage
	CPUs         int       `json:"cpus"`           // Number of pyhical or logical CPU cores
}

func CPU() (CPUStats, error) {
	cpuUsagePerCore, err := cpu.Percent(0, true)
	if err != nil {
		return CPUStats{}, fmt.Errorf("failed to get CPU usage per core: %w", err)
	}

	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return CPUStats{}, fmt.Errorf("failed to get total CPU usage: %w", err)
	}

	cpuCores, err := cpu.Counts(false)
	if err != nil {
		return CPUStats{}, fmt.Errorf("failed to get CPU cores: %w", err)
	}

	return CPUStats{
		UsagePerCore: cpuUsagePerCore,
		Usage:        cpuUsage,
		CPUs:         cpuCores,
	}, nil
}

type MemoryStats struct {
	Total       float64 `json:"total_gb"`     // Total memory in GB
	Available   float64 `json:"available_gb"` // Available memory in GB
	Used        float64 `json:"used_gb"`      // Used memory in GB
	UsedPercent float64 `json:"used_percent"` // Used memory percentage
	Free        float64 `json:"free_gb"`      // Free memory in GB
}

func Memory() (MemoryStats, error) {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return MemoryStats{}, fmt.Errorf("failed to get memory stats: %w", err)
	}

	return MemoryStats{
		Total:       float64(mem.Total) / 1024 / 1024 / 1024,
		Available:   float64(mem.Available) / 1024 / 1024 / 1024,
		Used:        float64(mem.Used) / 1024 / 1024 / 1024,
		UsedPercent: mem.UsedPercent,
		Free:        float64(mem.Free) / 1024 / 1024 / 1024,
	}, nil
}

type ProcessInfo struct {
	PID       int32    `json:"pid"`
	Name      string   `json:"name"`
	CPU       float64  `json:"cpu_usage"`
	Memory    float32  `json:"memory_usage_percent"`
	Arguments []string `json:"arguments"`
}

func Process() ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("getting processes: %w", err)
	}

	infos := make([]ProcessInfo, 0, len(procs))
	var info ProcessInfo
	for _, p := range procs {
		info, err = fetchProcessInfo(p)
		if err != nil {
			continue // skip uninspectable procs
		}
		infos = append(infos, info)
	}

	sort.SliceStable(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos, nil
}

func fetchProcessInfo(p *process.Process) (ProcessInfo, error) {
	name, err := p.Name()
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("getting process name: %w", err)
	}
	cpu, err := p.CPUPercent()
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("getting CPU percent: %w", err)
	}
	args, err := p.CmdlineSlice()
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("getting command line: %w", err)
	}
	mem, err := p.MemoryPercent()
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("getting memory percent: %w", err)
	}

	return ProcessInfo{
		PID:       p.Pid,
		Name:      name,
		CPU:       cpu,
		Arguments: args,
		Memory:    mem,
	}, nil
}
