#!/bin/bash

set -euo pipefail

IMAGES=(
    "roldyoran/go-client"
    "alpine"
)

TOTAL_CONTENEDORES=5
NOMBRE="p2-cronjob"
LOG_FILE="/var/log/cronjob.log"

log() {
    echo "$1" >> "$LOG_FILE"
}

random_container() {
    local commands=(
        "high-go-client"
        "high-alpine-cpu"
        "low-alpine-sleep"
    )

    local index=$((RANDOM % ${#commands[@]}))
    echo "${commands[$index]}"
}

create_containers() {
    local cmd="$1"
    local i="$2"
    local ts
    ts=$(date '+%Y%m%d%H%M%S')
    local container_name="${NOMBRE}-${ts}-${i}"

    log "Creando contenedor $container_name con comando $cmd"

    case "$cmd" in
        "high-go-client")
            docker run -d --name "$container_name" --label app=pr2-cronjob roldyoran/go-client >/dev/null
            ;;
        "high-alpine-cpu")
            docker run -d --name "$container_name" --label app=pr2-cronjob alpine sh -c "while true; do echo '2^20' | bc > /dev/null; sleep 1; done" >/dev/null
            ;;
        "low-alpine-sleep")
            docker run -d --name "$container_name" --label app=pr2-cronjob alpine sleep 240 >/dev/null
            ;;
    esac
}

main() {
    command -v docker >/dev/null 2>&1 || {
        log "ERROR: docker no encontrado"
        exit 1
    }

    log "Iniciando creación de $TOTAL_CONTENEDORES contenedores"

    for i in $(seq 1 "$TOTAL_CONTENEDORES"); do
        cmd=$(random_container)
        create_containers "$cmd" "$i"
    done

    log "Completado: $TOTAL_CONTENEDORES contenedores creados"
}

main "$@"
