# Zot HTTPS Registry Setup - Proyecto3 M.U.M.N.K8s

## Overview
Zot es un registro OCI HTTPS hospedado fuera del clúster K8s que almacena todas las imágenes Docker de los componentes del proyecto. Requiere HTTPS, autenticación basic, y soporte para descarga de OCI Artifacts.

**Carnet**: 201901271  
**Imágenes a publicar**:
- `rust-api:201901271`
- `api-client:201901271`
- `grpc-server:201901271`
- `go-consumer:201901271`

**OCI Artifact a publicar**: Configuración JSON con parámetros de RabbitMQ y Valkey

---

## Fase 1: Configuración de Zot en GCP VM (Fuera del Clúster)

### 1.1 Crear VM en GCP para Zot

```bash
# Crear VM with Ubuntu 24.04, 2 CPUs, 4GB RAM, 50GB disk
gcloud compute instances create zot-registry \
  --zone=us-central1-a \
  --machine-type=e2-medium \
  --image-family=ubuntu-2404-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=50GB \
  --tags=zot-registry \
  --metadata-from-file=startup-script=-

# Crear firewall rule para HTTPS
gcloud compute firewall-rules create allow-zot-https \
  --allow=tcp:5000 \
  --target-tags=zot-registry \
  --source-ranges=0.0.0.0/0

# Obtener IP pública (usarla como ZOT_HOST)
gcloud compute instances describe zot-registry \
  --zone=us-central1-a \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)'
```

### 1.2 Instalar Zot en VM

```bash
#!/bin/bash
# Script de startup para la VM

# Actualizar paquetes
sudo apt-get update && sudo apt-get install -y curl wget

# Descargar Zot última versión
sudo wget -O /usr/local/bin/zot https://github.com/project-zot/zot/releases/download/v2.1.16/zot-linux-amd64

# Hacer ejecutable
sudo chmod +x /usr/local/bin/zot

# Verificar instalación
zot --version
```

### 1.3 Generar Certificados HTTPS Autofirmados

```bash
# En la VM Zot:
mkdir -p /var/lib/zot/certs

# Generar certificado autofirmado válido 365 días
openssl req -x509 -newkey rsa:4096 -keyout /var/lib/zot/certs/key.pem \
  -out /var/lib/zot/certs/cert.pem -days 365 -nodes \
  -subj "/CN=zot.proyecto3.local/O=Proyecto3/C=GT"

# Verificar
openssl x509 -in /var/lib/zot/certs/cert.pem -text -noout | grep "Subject:\|Not Before\|Not After"
```

### 1.4 Crear Configuración de Zot (v2.1.16)

The configuration schema changed across Zot releases. Use the binary to validate your config after editing:

```bash
# Test config (runs Zot in foreground and prints schema errors)
sudo /usr/local/bin/zot serve /etc/zot/config.json
```

If you want a minimal, working HTTPS-enabled config for `v2.1.16` (no access-control policies), use the following example. Back up any existing `/etc/zot/config.json` first.

```json
{
  "storage": {
    "rootDirectory": "/var/lib/zot/storage"
  },
  "http": {
    "addr": "0.0.0.0:5000",
    "tls": {
      "cert": "/var/lib/zot/certs/cert.pem",
      "key": "/var/lib/zot/certs/key.pem"
    }
  },
  "auth": {
    "htpasswd": {
      "path": "/var/lib/zot/auth.htpasswd"
    }
  }
}
```

Notes:
- This minimal config starts Zot with HTTPS, basic htpasswd auth, and local filesystem storage.
- If you need advanced `access control` or `access log` configuration, copy the sample config from the Zot v2.1.16 release or the project documentation and adapt it. The schema names and casing must match the Zot binary; mismatches cause the service to fail with unmarshalling errors.
- After editing, validate with `sudo /usr/local/bin/zot serve /etc/zot/config.json` and check `sudo journalctl -u zot -f` for runtime errors.

### 1.5 Crear Archivo de Autenticación

```bash
# En la VM Zot:
sudo apt-get install -y apache2-utils

# Crear usuario admin con contraseña (ejemplo: admin123)
htpasswd -cb /var/lib/zot/auth.htpasswd admin admin123

# Crear directorio de almacenamiento
sudo mkdir -p /var/lib/zot/storage
sudo chmod 755 /var/lib/zot/storage

# Crear directorio de configuración
sudo mkdir -p /etc/zot
sudo cp config.json /etc/zot/config.json
```

### 1.6 Ejecutar Zot como Servicio Systemd

```bash
# /etc/systemd/system/zot.service
[Unit]
Description=Zot OCI Registry
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/zot
ExecStart=/usr/local/bin/zot serve /etc/zot/config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# Habilitar y iniciar
sudo systemctl daemon-reload
sudo systemctl enable zot
sudo systemctl start zot

# Verificar estado
sudo systemctl status zot
```

---

## Fase 2: Construir y Publicar Imágenes Docker

### 2.1 Configurar Credenciales Locales

```bash
# Guardar credenciales en ~/.docker/config.json
export ZOT_HOST="<IP_PUBLICA_VM>"  # Ejemplo: 35.192.12.45
export ZOT_USER="admin"
export ZOT_PASS="admin123"

# Login a Zot (ignorar HTTPS self-signed warning localmente)
docker login --username ${ZOT_USER} --password ${ZOT_PASS} ${ZOT_HOST}:5000
# O manualmente con flag inseguro:
DOCKER_CONTENT_TRUST=0 docker login -u admin -p admin123 ${ZOT_HOST}:5000
```

### 2.2 Construir Imágenes Localmente

```bash
cd /home/pdrpm/SOPES1/LAB_SO1_1S2026/proyecto3

# Rust API
cd api
docker build -t ${ZOT_HOST}:5000/api-rust:201901271 .
docker push ${ZOT_HOST}:5000/api-rust:201901271

# Go API Client
cd ../go-api-client
docker build \
  --build-context sharedpb=../go-shared-pb \
  -t ${ZOT_HOST}:5000/go-api-client:201901271 \
  .
docker push ${ZOT_HOST}:5000/go-api-client:201901271

# Go gRPC Server
cd ../go-grpc-server
cd go-grpc-server && docker build \
  --build-context sharedpb=../go-shared-pb \
  -t ${ZOT_HOST}:5000/go-grpc-server:201901271 \
  .
docker push ${ZOT_HOST}:5000/go-grpc-server:201901271

# Go Consumer
cd ../go-consumer
docker build -t ${ZOT_HOST}:5000/go-consumer:201901271 .
docker push ${ZOT_HOST}:5000/go-consumer:201901271
```

### 2.3 Verificar Imágenes en Zot

```bash
# Listar todas las imágenes en Zot
curl -k -u admin:admin123 https://${ZOT_HOST}:5000/v2/_catalog

# Listar tags de una imagen
curl -k -u admin:admin123 https://${ZOT_HOST}:5000/v2/api-rust/tags/list

# Ejemplo de salida:
# {
#   "name": "api-rust",
#   "tags": ["201901271"]
# }
```

---

## Fase 3: Crear y Publicar OCI Artifact

### 3.1 Crear Archivo de Configuración OCI Artifact

```json
# proyecto3-config.json (Contenido del artifact)
{
  "project": "M.U.M.N.K8s",
  "carnet": "201901271",
  "version": "1.0",
  "components": {
    "rust_api": {
      "port": 3000,
      "timeout_ms": 5000
    },
    "grpc_server": {
      "port": 50051,
      "timeout_ms": 10000
    },
    "rabbitmq": {
      "vhost": "/warreports",
      "persistence": true,
      "message_ttl_seconds": 604800,
      "durable_exchanges": true,
      "durable_queues": true
    },
    "valkey": {
      "port": 6379,
      "ttl_days": 7,
      "persistence": "rdb"
    },
    "grafana": {
      "port": 3000,
      "datasource": "valkey-svc.proyecto3.svc.cluster.local:6379"
    }
  },
  "deployment": {
    "namespace": "proyecto3",
    "hpa": {
      "min_replicas": 1,
      "max_replicas": 3,
      "cpu_threshold": 30,
      "memory_threshold": 70
    }
  }
}
```

### 3.2 Crear Manifest del Artifact

```bash
# Crear descriptor del artifact (OCI Image Spec)
cat > artifact-manifest.json <<EOF
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "size": $(wc -c < proyecto3-config.json),
    "digest": "sha256:$(sha256sum proyecto3-config.json | cut -d' ' -f1)"
  },
  "layers": [
    {
      "mediaType": "application/vnd.project3.config.v1+json",
      "size": $(wc -c < proyecto3-config.json),
      "digest": "sha256:$(sha256sum proyecto3-config.json | cut -d' ' -f1)"
    }
  ],
  "annotations": {
    "org.opencontainers.artifact.created": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')",
    "org.opencontainers.artifact.description": "Proyecto3 M.U.M.N.K8s Configuration",
    "carnet": "201901271"
  }
}
EOF
```

### 3.3 Publicar Artifact a Zot

```bash
# Usando ORAS (OCI Registry As Storage)
# Instalar ORAS si no está presente
wget https://github.com/oras-project/oras/releases/download/v1.1.0/oras_1.1.0_linux_amd64.tar.gz
tar xzf oras_1.1.0_linux_amd64.tar.gz
sudo mv oras /usr/local/bin/

# Publicar artifact
oras push ${ZOT_HOST}:5000/proyecto3-config:1.0 \
  --config artifact-manifest.json \
  proyecto3-config.json

# Verificar publicación
oras repo tags ${ZOT_HOST}:5000/proyecto3-config
```

---

## Fase 4: Integración con Kubernetes

### 4.1 Crear Secret para Zot Credentials

```bash
# En el clúster K8s:
kubectl create secret docker-registry zot-credentials \
  --docker-server=${ZOT_HOST}:5000 \
  --docker-username=admin \
  --docker-password=admin123 \
  --docker-email=proyecto3@local \
  -n proyecto3
```

### 4.2 Actualizar Manifests para Usar Zot

```yaml
# Ejemplo en deployment (ej: k8s/02-api-rust.yaml)
spec:
  template:
    spec:
      imagePullSecrets:
      - name: zot-credentials
      containers:
      - name: rust-api
        image: ${ZOT_HOST}:5000/api-rust:201901271
        imagePullPolicy: IfNotPresent
```

### 4.3 Descargar OCI Artifact en Pod Init

```yaml
# Ejemplo de init container que descarga artifact
initContainers:
- name: download-config
  image: ${ZOT_HOST}:5000/oras:latest
  command:
  - /bin/sh
  - -c
  - |
    oras pull ${ZOT_HOST}:5000/proyecto3-config:1.0 \
      -o /config
    cat /config/proyecto3-config.json
  volumeMounts:
  - name: config-volume
    mountPath: /config
volumes:
- name: config-volume
  emptyDir: {}
```

---

## Fase 5: Verificación y Troubleshooting

### 5.1 Verificar Conectividad a Zot

```bash
# Desde local:
curl -k -v https://${ZOT_HOST}:5000/v2/

# Esperado: 401 Unauthorized sin autenticación, 200 OK con credentials

# Desde dentro del clúster:
kubectl run -it debug --image=curlimages/curl -- \
  curl -k -u admin:admin123 https://${ZOT_HOST}:5000/v2/_catalog
```

### 5.2 Monitorear Logs de Zot

```bash
# En VM Zot:
sudo journalctl -u zot -f

# O directamente en archivo de log si está configurado
tail -f /var/log/zot/zot.log
```

### 5.3 Resolver Errores Comunes

| Error | Causa | Solución |
|-------|-------|----------|
| Certificate verify failed | HTTPS self-signed | Usar `curl -k` o importar cert |
| 401 Unauthorized | Credenciales incorrectas | Verificar htpasswd, login nuevamente |
| Connection refused | Zot no corriendo | `sudo systemctl restart zot` |
| ImagePullBackOff en Pod | Secret de credentials faltante | Crear secret con kubectl |
| HTTPS timeout | Firewall bloqueado | Verificar gcloud firewall-rules |

---

## Fase 6: Mantenimiento y Actualización

### 6.1 Limpiar Imágenes No Utilizadas

```bash
# En Zot:
zot garbage-collect /etc/zot/config.json

# O manualmente vía API:
curl -k -X DELETE -u admin:admin123 \
  https://${ZOT_HOST}:5000/v2/rust-api/manifests/sha256:abc123...
```

### 6.2 Renovar Certificados

```bash
# Antes de expiración:
openssl req -x509 -newkey rsa:4096 -keyout /var/lib/zot/certs/key.pem \
  -out /var/lib/zot/certs/cert.pem -days 365 -nodes \
  -subj "/CN=zot.proyecto3.local"

sudo systemctl restart zot
```

### 6.3 Backup del Registry

```bash
# Backup del storage
sudo tar czf /mnt/backups/zot-storage-$(date +%Y%m%d).tar.gz \
  /var/lib/zot/storage

# Backup de configuración
sudo tar czf /mnt/backups/zot-config-$(date +%Y%m%d).tar.gz \
  /etc/zot /var/lib/zot/certs
```

---

## Conclusión

Una vez completados estos pasos:
1. ✓ Zot HTTPS registry corriendo en GCP VM con certificado self-signed
2. ✓ Todas 4 imágenes Docker (carnet 201901271) publicadas
3. ✓ OCI Artifact (proyecto3-config.json) disponible para descarga
4. ✓ Secretos de autenticación creados en cluster K8s
5. ✓ Manifests actualizados para pull desde Zot

El proyecto cumple con los requisitos obligatorios de:
- **Zot**: Implementado en VM GCP fuera del clúster con HTTPS ✓
- **Todas imágenes publicadas**: rust-api, api-client, grpc-server, go-consumer ✓
- **OCI Artifact**: proyecto3-config.json descargable y usable en pods ✓
