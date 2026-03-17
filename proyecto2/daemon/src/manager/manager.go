package manager

import (
	"fmt"
	"log"
	"os/exec"
	"slices"
	"sort"

	"daemon-proyecto2/src/parser"
)

const (
	TARGET_LOW  = 3
	TARGET_HIGH = 2
)

// Contenedores protegidos que no deben ser eliminados por el daemon
var protectedContainers = []string{"grafana grafana", "valkey-server"}

// Manejar las políticas de gestión de contenedores según su uso (alto o bajo)
func Manage(containers []parser.ContainerInfo) error {
	// Filtrar contenedores protegidos
	var managed []parser.ContainerInfo
	for _, c := range containers {
		if !isProtected(c.Name) {
			managed = append(managed, c)
		}
	}

	// Dividir en grupos de alto y bajo consumo
	high, low := splitByUsage(managed)

	log.Printf("[manager] High usage containers: %d (target: %d)", len(high), TARGET_HIGH)
	log.Printf("[manager] Low usage containers: %d (target: %d)", len(low), TARGET_LOW)

	// Ordenar cada grupo por uso de recursos (RAM, VSZ, RSS, CPU) para eliminar los más costosos primero
	sortContainers(high)
	sortContainers(low)

	// Manejar contenedores de alto consumo
	if err := enforceLimit(high, TARGET_HIGH, "high"); err != nil {
		return err
	}

	// Manejar contenedores de bajo consumo
	if err := enforceLimit(low, TARGET_LOW, "low"); err != nil {
		return err
	}

	return nil
}

// Eliminar contenedores excedentes para cumplir con el objetivo, priorizando los de mayor consumo
func enforceLimit(containers []parser.ContainerInfo, target int, label string) error {
	if len(containers) <= target {
		return nil
	}

	// Remover los contenedores excedentes, dejando solo los más costosos
	toRemove := containers[:len(containers)-target]

	for _, c := range toRemove {
		log.Printf("[manager] Removing %s container: %s (RAM=%dKB VSZ=%dKB RSS=%dKB CPU=%.2f%%)",
			label, c.Name, c.RAM, c.VSZ, c.RSS, c.CPU)

		if err := stopAndRemove(c.ID); err != nil {
			log.Printf("[manager] Warning: failed to remove %s: %v", c.Name, err)
		}
	}

	return nil
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
	return slices.Contains(protectedContainers, name)
}
