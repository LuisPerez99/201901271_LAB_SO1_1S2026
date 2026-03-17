package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ContainerInfo struct {
	ID        string
	Name      string
	Command   string
	PID       int
	RAM       uint64  // Compatible metric for sorting/storage; mirrors RSS from /proc.
	VSZ       uint64  // Virtual memory size in KB
	RSS       uint64  // Physical memory in KB
	MemoryPct float64 // Memory usage percentage
	CPU       float64 // CPU usage percentage
	IsHigh    bool    // true = high usage, false = low usage
}

// Leer la información de los contenedores desde el archivo /proc/continfo_pr2_so1_201901271
func ReadContainerInfo(path string) ([]ContainerInfo, error) {
	file, error := os.Open(path)
	if error != nil {
		return nil, fmt.Errorf("error al abrir el archivo: %v", error)
	}
	defer file.Close()

	var containers []ContainerInfo

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "Memoria total:") || strings.HasPrefix(line, "PID\t") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		vsz, err := strconv.ParseUint(fields[3], 10, 64)
		if err != nil {
			continue
		}

		rss, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			continue
		}

		memPct, err := parsePercent(fields[5])
		if err != nil {
			memPct = 0
		}

		cpu, err := parsePercent(fields[6])
		if err != nil {
			cpu = 0
		}

		container := ContainerInfo{
			ID:        fields[0],
			Name:      fields[1],
			Command:   fields[2],
			PID:       pid,
			RAM:       rss,
			VSZ:       vsz,
			RSS:       rss,
			MemoryPct: memPct,
			CPU:       cpu,
		}

		containers = append(containers, container)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al leer el archivo: %v", err)
	}

	return containers, nil
}

func parsePercent(value string) (float64, error) {
	clean := strings.TrimSuffix(strings.TrimSpace(value), "%")
	return strconv.ParseFloat(clean, 64)
}
