package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"time"

	"daemon-proyecto2/src/db"
	"daemon-proyecto2/src/parser"
)

const (
	TARGET_LOW  = 3
	TARGET_HIGH = 2
)

const cronJobLabel = "app=pr2-cronjob"

// Contenedores protegidos que no deben ser eliminados por el daemon
var protectedContainers = []string{"grafana", "valkey"}

type dockerInspect struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Pid int `json:"Pid"`
	} `json:"State"`
	Config struct {
		Image  string            `json:"Image"`
		Cmd    []string          `json:"Cmd"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

// Manejar las políticas de gestión de contenedores según su uso (alto o bajo)
func Manage(containers []parser.ContainerInfo) ([]parser.ContainerInfo, *db.DeletionStats, error) {
	dockerContainers, err := resolveManagedContainers(containers)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[manager] Contenedores Docker encontrados desde /proc: %d", len(dockerContainers))

	// Dividir en grupos de alto y bajo consumo
	high, low := splitByUsage(dockerContainers)

	log.Printf("[manager] Contenedores de alto consumo: %d (objetivo: %d)", len(high), TARGET_HIGH)
	log.Printf("[manager] Contenedores de bajo consumo: %d (objetivo: %d)", len(low), TARGET_LOW)

	// Ordenar cada grupo por uso de recursos (RAM, VSZ, RSS, CPU) para eliminar los más costosos primero
	sortContainers(high)
	sortContainers(low)

	stats := &db.DeletionStats{
		Timestamp: time.Now().Unix(),
		Deleted:   make([]map[string]interface{}, 0),
	}

	// Manejar contenedores de alto consumo
	if err := enforceLimit(high, TARGET_HIGH, "high", stats); err != nil {
		return nil, nil, err
	}

	// Manejar contenedores de bajo consumo
	if err := enforceLimit(low, TARGET_LOW, "low", stats); err != nil {
		return nil, nil, err
	}

	remaining, err := resolveManagedContainers(containers)
	if err != nil {
		return nil, nil, err
	}

	return remaining, stats, nil
}

// Eliminar contenedores excedentes para cumplir con el objetivo, priorizando los de mayor consumo
func enforceLimit(containers []parser.ContainerInfo, target int, label string, stats *db.DeletionStats) error {
	if len(containers) <= target {
		return nil
	}

	// Remover los contenedores excedentes, dejando solo los más costosos
	toRemove := containers[:len(containers)-target]

	for _, c := range toRemove {
		log.Printf("[manager] Eliminando contenedor %s: %s (RAM=%dKB VSZ=%dKB RSS=%dKB CPU=%.2f%%)",
			label, c.Name, c.RAM, c.VSZ, c.RSS, c.CPU)

		if err := stopAndRemove(c.ID); err != nil {
			log.Printf("[manager] Advertencia: no se pudo eliminar %s: %v", c.Name, err)
			continue
		}

		entry := map[string]interface{}{
			"name": c.Name,
			"type": label,
			"ram":  c.RAM,
			"vsz":  c.VSZ,
			"rss":  c.RSS,
			"cpu":  c.CPU,
		}
		stats.Deleted = append(stats.Deleted, entry)
		stats.TotalCount++
		if label == "high" {
			stats.HighRemoved++
		} else {
			stats.LowRemoved++
		}
	}

	return nil
}

func resolveManagedContainers(processes []parser.ContainerInfo) ([]parser.ContainerInfo, error) {
	inspected, err := listCronJobContainers()
	if err != nil {
		return nil, err
	}

	byPID := make(map[int]dockerInspect, len(inspected))
	for _, container := range inspected {
		if container.State.Pid <= 0 {
			continue
		}
		byPID[container.State.Pid] = container
	}

	managed := make([]parser.ContainerInfo, 0, len(processes))
	for _, proc := range processes {
		container, ok := byPID[proc.PID]
		if !ok {
			continue
		}

		name := strings.TrimPrefix(container.Name, "/")
		if isProtected(name) {
			continue
		}

		proc.ID = container.ID
		proc.Name = name
		proc.IsHigh = isHighUsageContainer(container)
		managed = append(managed, proc)
	}

	return managed, nil
}

func listCronJobContainers() ([]dockerInspect, error) {
	psCmd := exec.Command("docker", "ps", "-aq", "--filter", "label="+cronJobLabel)
	out, err := psCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return nil, nil
	}

	args := append([]string{"inspect"}, ids...)
	inspectCmd := exec.Command("docker", args...)
	inspectOut, err := inspectCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var containers []dockerInspect
	if err := json.Unmarshal(inspectOut, &containers); err != nil {
		return nil, fmt.Errorf("error parsing docker inspect output: %w", err)
	}

	return containers, nil
}

func RemoveAllCronJobContainers() error {
	containers, err := listCronJobContainers()
	if err != nil {
		return err
	}

	for _, container := range containers {
		name := strings.TrimPrefix(container.Name, "/")
		if isProtected(name) {
			continue
		}

		log.Printf("[manager] Eliminando contenedor del cronjob al apagar: %s", name)
		if err := stopAndRemove(container.ID); err != nil {
			log.Printf("[manager] Advertencia: no se pudo eliminar %s durante el apagado: %v", name, err)
		}
	}

	return nil
}

func isHighUsageContainer(container dockerInspect) bool {
	if strings.Contains(container.Config.Image, "roldyoran/go-client") {
		return true
	}

	cmd := strings.Join(container.Config.Cmd, " ")
	if strings.Contains(cmd, "sleep 240") {
		return false
	}

	return true
}

// Ordenar contenedores por uso de recursos (RAM, VSZ, RSS, CPU) de mayor a menor
func sortContainers(containers []parser.ContainerInfo) {
	sort.Slice(containers, func(i, j int) bool {
		a, b := containers[i], containers[j]

		if a.RAM != b.RAM {
			return a.RAM > b.RAM
		}
		if a.VSZ != b.VSZ {
			return a.VSZ > b.VSZ
		}
		if a.RSS != b.RSS {
			return a.RSS > b.RSS
		}
		return a.CPU > b.CPU
	})
}

// Separar contenedores en dos grupos: alto consumo y bajo consumo, basado en la clasificación del parser
func splitByUsage(containers []parser.ContainerInfo) (high, low []parser.ContainerInfo) {
	for _, c := range containers {
		if c.IsHigh {
			high = append(high, c)
		} else {
			low = append(low, c)
		}
	}
	return
}

// Detener y eliminar un contenedor por su ID usando comandos de Docker
func stopAndRemove(id string) error {
	stopCmd := exec.Command("docker", "stop", id)
	if out, err := stopCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker stop %s failed: %w\n%s", id, err, string(out))
	}

	rmCmd := exec.Command("docker", "rm", id)
	if out, err := rmCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm %s failed: %w\n%s", id, err, string(out))
	}

	return nil
}

// Verificar si un contenedor está en la lista de protegidos para evitar su eliminación por el daemon
func isProtected(name string) bool {
	for _, protected := range protectedContainers {
		if name == protected {
			return true
		}
	}
	return false
}
