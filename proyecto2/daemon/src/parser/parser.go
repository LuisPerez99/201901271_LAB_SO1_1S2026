package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ContainerInfo struct {
	ID     string
	Name   string
	PID    int
	RAM    uint64  // RAM usage in KB
	VSZ    uint64  // Virtual memory size in KB
	RSS    uint64  // Physical memory in KB
	CPU    float64 // CPU usage percentage
	IsHigh bool    // true = high usage, false = low usage
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

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		pid, _ := strconv.Atoi(fields[2])
		ram, _ := strconv.ParseUint(fields[3], 10, 64)
		vsz, _ := strconv.ParseUint(fields[4], 10, 64)
		rss, _ := strconv.ParseUint(fields[5], 10, 64)
		cpu, _ := strconv.ParseFloat(fields[6], 64)

		container := ContainerInfo{
			ID:   fields[0],
			Name: fields[1],
			PID:  pid,
			RAM:  ram,
			VSZ:  vsz,
			RSS:  rss,
			CPU:  cpu,
		}

		// Clasificar basado en el nombre del contenedor
		container.IsHigh = isHighUsage(container.Name)
		containers = append(containers, container)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al leer el archivo: %v", err)
	}

	return containers, nil
}

// Determinar si el contenedor es de alto o bajo consumo basado en su nombre
func isHighUsage(name string) bool {
	return strings.Contains(name, "go-client") || strings.Contains(name, "high")
}
