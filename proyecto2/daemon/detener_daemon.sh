#!/bin/bash

echo "Deteniendo daemon..."

if [ -f daemon.pid ]; then
    PID=$(cat daemon.pid)
    if kill -0 "$PID" >/dev/null 2>&1; then
        sudo kill "$PID"
        echo "Daemon detenido con PID $PID"
    else
        echo "No se encontró un proceso activo con PID $PID. Eliminando archivo PID."
    fi
    rm -f daemon.pid
else
    echo "Archivo daemon.pid no encontrado. ¿Está el daemon corriendo?"
fi
