# Manual Técnico

## Proyecto 1 - Sistemas Operativos 1

**Nombre:** Luis Pedro Pérez Monzón

**Carné:** 201901271

### Diagrama de Arquitectura

![Diagrama de Arquitectura](Imagenes/arquitectura.png)

### Funcionamiento de las APIs

#### Endpoints

**`GET /health`**

- Descripción: Verifica el estado de la API.
- Respuesta:

    ```json
    {
        "status": "UP",
        "message": "API# is Ready",
        "timestamp": "2026-02-16T12:00:00Z",
        "vm": "1",
        "carnet": "201901271"
    }
    ```

**`GET /api#/201901271/call-api#`**

- Descripción: Llama a la API y devuelve un mensaje con la respuesta del servidor.
- Respuesta:

    ```json
    {
        "status": "UP",
        "message": "API# is Ready",
        "timestamp": "2026-02-16T17:13:41Z",
        "vm": "1",
        "carnet": "201901271"
    }
    ```

### Configuración de Máquinas Virtuales

**SO:** Ubuntu Server 24.04 LTS
**CPU:** 2 vCPU  
**RAM:** 2 GB  
**Disco Duro:** 15 GB

![Configuración de Máquinas Virtuales](Imagenes/vm_config_1.png)

![Configuración de Máquinas Virtuales](Imagenes/vm_config_2.png)

![Configuración de Máquinas Virtuales](Imagenes/vm_config_3.png)

![Configuración de Máquinas Virtuales](Imagenes/vm_config_4.png)
