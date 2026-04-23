**M.U.M.N.K8s - Monitoreo de Unidades Militares en la Nube con Kubernetes**

Proyecto distribuido escalable que simula el procesamiento en tiempo real de reportes militares.

## Estructura de Directorios

```
proyecto3/
├── api/                    # Servicio REST en Rust
│   ├── src/
│   │   ├── main.rs        # Entry point
│   │   ├── models.rs      # Estructuras de datos
│   │   ├── handlers.rs    # Manejadores HTTP
│   │   └── config.rs      # Configuración
│   ├── Cargo.toml
│   └── Dockerfile
├── go-api-client/         # Go Service 1: REST proxy + gRPC client
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── go-grpc-server/        # Go Service 2: gRPC server + RabbitMQ writer
│   ├── server/
│   │   └── server.go
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── go-consumer/           # Go Consumer: RabbitMQ consumer → Valkey
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── proto/                 # gRPC Protocol Buffer definitions
│   └── warreport.proto
├── k8s/                   # Kubernetes manifests
│   ├── 01-namespace.yaml
│   ├── 02-rust-api.yaml
│   ├── 03-rabbitmq.yaml
│   ├── 04-go-api-client.yaml
│   ├── 05-go-grpc-server.yaml
│   ├── 06-go-consumer.yaml
│   ├── 07-hpa.yaml
│   ├── 08-gateway-api.yaml
│   ├── 09-kubevirt-valkey.yaml
│   └── 10-kubevirt-grafana.yaml
├── locust/                # Load generation scripts
│   ├── locustfile.py
│   └── requirements.txt
├── docs/                  # Documentation
│   ├── MANUAL_TECNICO.md
│   ├── ZOT_REGISTRY_SETUP.md  # Zot HTTPS registry setup guide
│   └── DEPLOYMENT_VERIFICATION.md  # Post-deployment validation
└── README.md (this file)
```

## Componentes

1. **Rust API REST** - Entry point, recibe solicitudes de Locust
2. **Go Service 1** - REST proxy + gRPC client, desplegado con 2 contenedores
3. **Go Service 2** - gRPC server + RabbitMQ publisher, desplegado con 2 contenedores
4. **RabbitMQ** - Message broker
5. **Go Consumer** - Consume mensajes → almacena en Valkey
6. **Valkey** - Base de datos en memoria (en KubeVirt VM)
7. **Grafana** - Visualización (en KubeVirt VM)

## Flujo de Datos

```
Locust (JSON)
    ↓
[Gateway API] /grpc-201901271
    ↓
Rust API (REST) :3000
    ↓
Go Service 1 (REST client + gRPC)
    ↓
Go Service 2 (gRPC server)
    ↓
RabbitMQ (Topic: warreports)
    ↓
Go Consumer (subscribe to reports.usa)
    ↓
Valkey ← datos persisten
    ↓
Grafana (visualización)
```

## Inicio Rápido

### 0. Setup Zot HTTPS Registry (REQUERIDO)

Consultar: [docs/ZOT_REGISTRY_SETUP.md](docs/ZOT_REGISTRY_SETUP.md)

```bash
# En GCP VM:
# 1. Instalar Zot con HTTPS
# 2. Generar certificados autofirmados  
# 3. Crear credentials: htpasswd admin:admin123
# 4. Ejecutar como systemd service

# Localmente:
export ZOT_HOST="<IP_VM_GCP>"
docker login -u admin -p admin123 ${ZOT_HOST}:5000

# Construir y publicar todas las imágenes
./scripts/build-and-push-images.sh ${ZOT_HOST}:5000

# Verificar
curl -k -u admin:admin123 https://${ZOT_HOST}:5000/v2/_catalog
```

### 1. Deploy Rust API
```bash
cd api
docker build -t zot:5000/api-rust:v1 .
docker push zot:5000/api-rust:v1
kubectl apply -f k8s/rust-api.yaml
```

### 2. Deploy Go Services
```bash
# Service 1
cd go-api-client
docker build -t zot:5000/go-api-client:v1 .
docker push zot:5000/go-api-client:v1

# Service 2
cd ../go-grpc-server
docker build -t zot:5000/go-grpc-server:v1 .
docker push zot:5000/go-grpc-server:v1

kubectl apply -f k8s/go-api-client.yaml
kubectl apply -f k8s/go-grpc-server.yaml
```

### 3. Deploy RabbitMQ y Consumer
```bash
kubectl apply -f k8s/rabbitmq.yaml
kubectl apply -f k8s/consumer.yaml
```

### 4. Gateway API
```bash
kubectl apply -f k8s/gateway-api.yaml
```

### 5. Testing
```bash
# Port-forward a Rust API
kubectl port-forward svc/rust-api 3000:3000

# Enviar reporte de prueba
curl -X POST http://localhost:3000/report \
  -H "Content-Type: application/json" \
  -d '{
    "country": "USA",
    "warplanes_in_air": 25,
    "warships_in_water": 12,
    "timestamp": "2026-04-14T10:30:00Z"
  }'
```

## Status

- [x] Structure created
- [x] Proto file defined
- [x] Rust API skeleton
- [ ] Go Services implementation
- [ ] RabbitMQ deployment
- [ ] Consumer implementation
- [x] Valkey setup
- [x] Grafana setup
- [ ] Load testing
- [ ] Documentation

## Timeline

- **Semana 1**: Componentes base (Rust + Go)
- **Semana 2**: Mensajería + Almacenamiento
- **Semana 3**: Pruebas + HPA
- **Semana 4**: Documentación + Dapr (opcional)
