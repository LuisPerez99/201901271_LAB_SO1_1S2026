package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var memoryLineRegex = regexp.MustCompile(`Memoria total:\s*(\d+)\s*MB,\s*Memoria libre:\s*(\d+)\s*MB,\s*Memoria usada:\s*(\d+)\s*MB`)

type SystemMemoryInfo struct {
	TotalMB uint64 `json:"total_mb"`
	FreeMB  uint64 `json:"free_mb"`
	UsedMB  uint64 `json:"used_mb"`
}

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

func ReadSystemMemoryInfo(path string) (SystemMemoryInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return SystemMemoryInfo{}, fmt.Errorf("error al abrir el archivo: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Memoria total:") {
			continue
		}

		matches := memoryLineRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			return SystemMemoryInfo{}, fmt.Errorf("formato de memoria inválido: %s", line)
		}

		totalMB, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			return SystemMemoryInfo{}, fmt.Errorf("error parseando memoria total: %w", err)
		}

		freeMB, err := strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			return SystemMemoryInfo{}, fmt.Errorf("error parseando memoria libre: %w", err)
		}

		usedMB, err := strconv.ParseUint(matches[3], 10, 64)
		if err != nil {
			return SystemMemoryInfo{}, fmt.Errorf("error parseando memoria usada: %w", err)
		}

		return SystemMemoryInfo{
			TotalMB: totalMB,
			FreeMB:  freeMB,
			UsedMB:  usedMB,
		}, nil
	}

	if err := scanner.Err(); err != nil {
		return SystemMemoryInfo{}, fmt.Errorf("error al leer el archivo: %v", err)
	}

	return SystemMemoryInfo{}, fmt.Errorf("no se encontró la línea de memoria en %s", path)
}

func parsePercent(value string) (float64, error) {
	clean := strings.TrimSuffix(strings.TrimSpace(value), "%")
	return strconv.ParseFloat(clean, 64)
}
