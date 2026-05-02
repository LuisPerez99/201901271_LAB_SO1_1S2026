# Guía de Deployment M.U.M.N.K8s

## Checklist de Pre-deployment

- [ ] Cluster GKE disponible con N1 instances
- [ ] KubeVirt instalado en el cluster
- [ ] Zot registry configurado con HTTPS
- [ ] Credenciales de Zot disponibles
- [ ] git clone del repositorio

## Fase 1: Build y Push de Imágenes

### IP del registro ZOT
```bash
ZOT_HOST=<IP_VM>
```

### Rust API
```bash
cd proyecto3/api
docker build -t rust-api:201901271 .
# Tag y push a Zot
docker tag rust-api:201901271 ${ZOT_HOST}:5000/rust-api:201901271
docker push ${ZOT_HOST}:5000/rust-api:201901271
```

### Go API Client
```bash
cd proyecto3/go-api-client
docker build \
  --build-context sharedpb=../go-shared-pb \
  -t go-api-client:201901271 \
  .
docker tag go-api-client:201901271 ${ZOT_HOST}:5000/go-api-client:201901271
docker push ${ZOT_HOST}:5000/go-api-client:201901271
```

### Go gRPC Server
```bash
cd proyecto3/go-grpc-server
docker build \
  --build-context sharedpb=../go-shared-pb \
  -t ${ZOT_HOST}:5000/go-grpc-server:201901271 \
  .
docker push ${ZOT_HOST}:5000/go-grpc-server:201901271
```

### Go Consumer
```bash
cd proyecto3/go-consumer
docker build -t ${ZOT_HOST}:5000/go-consumer:201901271 .
docker push ${ZOT_HOST}:5000/go-consumer:201901271
```

## Fase 2: Deploy a Kubernetes

### Crear namespace
```bash
kubectl apply -f k8s/01-namespace.yaml
```

### Deploy componentes base
```bash
kubectl apply -f k8s/02-rust-api.yaml
kubectl apply -f k8s/03-rabbitmq.yaml
kubectl apply -f k8s/04-go-api-client.yaml
kubectl apply -f k8s/05-go-grpc-server.yaml
kubectl apply -f k8s/06-go-consumer.yaml
```

### Verificar estado
```bash
kubectl get pods -n proyecto3
kubectl get svc -n proyecto3
```

### Deploy RabbitMQ con persistencia
```bash
kubectl apply -f k8s/03-rabbitmq.yaml
# Esto crea: Deployment, Service y PersistentVolumeClaim
```

### Deploy Gateway API
```bash
kubectl apply -f k8s/08-gateway-api.yaml
```

### Deploy KubeVirt VMs
```bash
kubectl apply -f k8s/09-kubevirt-valkey.yaml
kubectl apply -f k8s/10-kubevirt-grafana.yaml
```

### Verificar servicios de VM
```bash
kubectl get vm -n proyecto3
kubectl get svc -n proyecto3 | grep -E 'valkey-svc|grafana-svc'
```

### Deploy HPA 
```bash
kubectl apply -f k8s/07-hpa.yaml
```

## Fase 3: Testing

### Obtener IP del Gateway
```bash
kubectl get gateway -n proyecto3
kubectl get svc -A | grep gateway
```

### Ruta obligatoria del proyecto
- Endpoint público requerido: `/grpc-201901271`

### Port-forward a Rust API
```bash
kubectl port-forward svc/rust-api -n proyecto3 3000:3000
```

### Enviar reporte de prueba
```bash
curl -X POST http://localhost:3000/report \
  -H "Content-Type: application/json" \
  -d '{
    "country": "USA",
    "warplanes_in_air": 25,
    "warships_in_water": 12,
    "timestamp": "2026-04-14T10:30:00Z"
  }'
```

### Probar ruta de Gateway requerida
```bash
curl -X POST http://<GATEWAY_IP>/grpc-201901271 \
  -H "Content-Type: application/json" \
  -d '{
    "country": "USA",
    "warplanes_in_air": 25,
    "warships_in_water": 12,
    "timestamp": "2026-04-14T10:30:00Z"
  }'
```

## Troubleshooting

### Ver logs de un pod
```bash
kubectl logs -n proyecto3 <pod-name> -f
```

### Shell en un pod
```bash
kubectl exec -it -n proyecto3 <pod-name> -- /bin/sh
```

### Checking RabbitMQ
```bash
kubectl port-forward -n proyecto3 svc/rabbitmq 15672:15672
# Acceder a http://localhost:15672 (guest/guest)
```

### Checking Valkey VM
```bash
kubectl port-forward -n proyecto3 svc/valkey-svc 6379:6379
```

### Verificar claves del Consumer en Valkey
```bash
redis-cli -h 127.0.0.1 -p 6379 KEYS "dashboard:*"
redis-cli -h 127.0.0.1 -p 6379 GET "dashboard:global:reports_total"
redis-cli -h 127.0.0.1 -p 6379 KEYS "dashboard:country:*"
```

### Checking Grafana VM
```bash
kubectl port-forward -n proyecto3 svc/grafana-svc 3000:3000
```

### Verificar datasource de Grafana
- Datasource provisionado automáticamente: `Valkey`
- Endpoint interno usado por Grafana: `valkey-svc.proyecto3.svc.cluster.local:6379`
- Ingreso local por port-forward: `http://127.0.0.1:3000`

### Restart deployment
```bash
kubectl rollout restart deployment/<name> -n proyecto3
```

## Próximos pasos

1. [ ] Completar implementación de gRPC en Go services
2. [ ] Setup KubeVirt VMs (Valkey + Grafana)
3. [ ] Conectar Grafana a Valkey
4. [ ] Load testing con Locust
5. [ ] Análisis y documentación

## Resumen de Sesión

### Estado actual
- Rust API: validación de reportes y reenvío HTTP a Go API Client.
- Go API Client: deployment con 2 contenedores, recibe reportes y llama al servicio gRPC.
- Go gRPC Server: deployment con 2 contenedores, procesa `SendReport` y publica en RabbitMQ.
- Go Consumer: consume de RabbitMQ y persiste en Valkey con claves agregadas para dashboard.
- Valkey VM: expuesto como `valkey-svc` dentro de KubeVirt.
- Grafana VM: expuesto como `grafana-svc` dentro de KubeVirt, con datasource Valkey provisionado.
- Protobuf compartido: generado en `go-shared-pb/`.
- Kubernetes: manifests base creados para API, RabbitMQ, Gateway API y HPA.

### Validación realizada
- `cargo check` para la API Rust.
- `go build ./...` en `go-shared-pb`, `go-api-client`, `go-grpc-server` y `go-consumer`.

### Dependencias pendientes
1. Levantar Valkey en KubeVirt.
2. Levantar Grafana en KubeVirt.
3. Probar el flujo completo en GKE con Gateway API.
4. Ejecutar Locust y ajustar HPA.
5. Completar el manual técnico final.

### Cómo continuar en otra máquina
1. Clona el repositorio.
2. Ejecuta `cargo check` en `proyecto3/api`.
3. Ejecuta `go build ./...` en `go-shared-pb`, `go-api-client`, `go-grpc-server` y `go-consumer`.
4. Aplica los manifests de `k8s/` en el cluster.
5. Continúa con Valkey, Grafana y las pruebas de carga.

### Checklist corto de arranque
- [ ] `git clone` del repositorio.
- [ ] `cargo check` en `proyecto3/api`.
- [ ] `go build ./...` en `go-shared-pb`.
- [ ] `go build ./...` en `go-api-client`.
- [ ] `go build ./...` en `go-grpc-server`.
- [ ] `go build ./...` en `go-consumer`.
- [ ] `kubectl apply -f k8s/01-namespace.yaml`.
- [ ] `kubectl apply -f k8s/02-rust-api.yaml`.
- [ ] `kubectl apply -f k8s/03-rabbitmq.yaml`.
- [ ] `kubectl apply -f k8s/04-go-api-client.yaml`.
- [ ] `kubectl apply -f k8s/05-go-grpc-server.yaml`.
- [ ] `kubectl apply -f k8s/06-go-consumer.yaml`.
- [ ] `kubectl apply -f k8s/08-gateway-api.yaml`.
