# Requisitos — kubefin

## Introducción

`kubefin` es una CLI en Go que unifica el diagnóstico de clústeres EKS con el análisis de costos de AWS (FinOps), exponiendo su lógica a través de tres superficies: CLI, servidor MCP y agente de IA (Groq). Reemplaza a `EKS-resources-review` (Python) y `scripts-sh`. El proyecto prioriza operar íntegramente sobre tiers gratuitos y mantener toda operación como read-only en la versión 1.

Los requisitos usan notación EARS (Easy Approach to Requirements Syntax).

## Glosario

- **CUR**: Cost & Usage Report — export detallado de costos de AWS a un bucket S3.
- **CE API**: Cost Explorer API — API programática de costos, cobra $0.01 por request paginado.
- **Tool**: unidad atómica de capacidad, reutilizable desde CLI, MCP y agente.
- **Superficie**: forma de invocar las tools (CLI, MCP, agente).

## Requisitos

### R1 — Diagnóstico de EKS vía client-go

**Historia:** Como SRE, quiero diagnosticar el estado de mi clúster EKS sin depender de parsear salida de `kubectl`, para obtener datos fiables y tipados.

#### Criterios de aceptación

1. CUANDO el usuario ejecuta un comando de diagnóstico EKS, el sistema DEBERÁ obtener los datos mediante `client-go` contra la API de Kubernetes, no mediante shell-out a `kubectl`.
2. CUANDO no exista un contexto de kubectl válido, el sistema DEBERÁ fallar con un mensaje claro y código de salida distinto de cero.
3. El sistema DEBERÁ cubrir como mínimo: estado de nodos, uso de recursos por nodo, top consumidores, recursos por namespace, quotas, reinicios de pods, requests/limits vs uso real, configuración y errores de Karpenter, cobertura de Datadog Agent, estado de componentes core de EKS, y recomendaciones.
4. CUANDO Metrics Server no esté disponible, el sistema DEBERÁ informarlo sin abortar el resto del diagnóstico.
5. CUANDO el usuario solicite un reporte completo, el sistema DEBERÁ poder exportarlo a Markdown, JSON y Confluence Wiki Markup.

### R2 — Análisis de costos gratuito (CUR-first)

**Historia:** Como dueño de una cuenta AWS personal en free-tier, quiero analizar mis costos sin incurrir en cargos, para mantenerme dentro del tier gratuito.

#### Criterios de aceptación

1. POR DEFECTO, el sistema DEBERÁ obtener los datos de costo leyendo el CUR desde S3, sin invocar la CE API.
2. CUANDO el usuario solicite datos que requieran la CE API, el sistema DEBERÁ requerir el flag explícito `--use-ce-api` Y advertir del costo de $0.01 por request antes de proceder.
3. SI el flag `--use-ce-api` no está presente y los datos solicitados solo son obtenibles vía CE API, ENTONCES el sistema DEBERÁ rechazar la operación con un mensaje explicativo.
4. El sistema DEBERÁ soportar análisis de costo agrupado por servicio, por tag y por periodo (últimos N días/meses).
5. CUANDO el CUR aún no esté disponible (cuenta nueva, sin export configurado), el sistema DEBERÁ indicar cómo configurarlo y NO fallar silenciosamente.

### R3 — Recomendaciones de rightsizing (pod a factura)

**Historia:** Como platform engineer, quiero correlacionar el uso real de recursos de mis pods con las recomendaciones de rightsizing de la infraestructura y el costo real, para optimizar el gasto de mi clúster.

#### Criterios de aceptación

1. CUANDO el usuario solicite recomendaciones, el sistema DEBERÁ obtener rightsizing de Compute Optimizer (gratis) para EC2/EBS.
2. El sistema DEBERÁ correlacionar el análisis de requests/limits vs uso (de R1) con las recomendaciones de infraestructura.
3. CUANDO existan datos de costo del CUR, el sistema DEBERÁ estimar el ahorro potencial de cada recomendación.
4. Cada recomendación DEBERÁ incluir severidad, evidencia, acción sugerida y ahorro estimado cuando aplique.

### R4 — Superficie MCP

**Historia:** Como usuario de un cliente LLM (Kiro, Claude Desktop), quiero consultar mis tools de kubefin desde el asistente, para obtener diagnósticos y costos sin cambiar de contexto.

#### Criterios de aceptación

1. CUANDO el usuario ejecuta `kubefin mcp`, el sistema DEBERÁ iniciar un servidor MCP por stdio usando el SDK oficial de Go.
2. El servidor MCP DEBERÁ exponer las mismas tools core que usa la CLI, sin duplicar lógica.
3. Cada tool MCP DEBERÁ declarar su esquema de entrada y salida.
4. CUANDO una tool MCP realice una operación que pueda generar costo, el sistema DEBERÁ reflejar la misma política de opt-in que la CLI (R2).

### R5 — Agente de IA con Groq

**Historia:** Como ingeniero, quiero preguntar en lenguaje natural por qué cambió mi costo o el estado de mi clúster, para obtener un diagnóstico correlacionado sin ejecutar comandos manualmente.

#### Criterios de aceptación

1. CUANDO el usuario ejecuta `kubefin ask "<pregunta>"`, el sistema DEBERÁ usar Groq con tool-use para invocar las tools core iterativamente hasta responder.
2. SI la variable `GROQ_API_KEY` no está definida, ENTONCES el sistema DEBERÁ fallar con un mensaje que indique cómo obtenerla.
3. El agente DEBERÁ devolver hallazgos estructurados (severidad, título, evidencia, recomendación, ahorro estimado), no prosa libre sin estructura.
4. El agente NUNCA DEBERÁ ejecutar cambios de infraestructura ni operaciones de escritura en la versión 1.
5. CUANDO el proveedor de IA sea remoto, el sistema DEBERÁ ofrecer redacción de datos sensibles (account IDs, ARNs, IPs internas) antes del envío.
6. El proveedor de IA DEBERÁ ser intercambiable mediante una interfaz, con Groq como implementación por defecto.

### R6 — Operación gratuita y guardarraíles de costo

**Historia:** Como dueño de la cuenta, quiero que la herramienta no genere cargos inesperados, para confiar en usarla libremente.

#### Criterios de aceptación

1. El sistema DEBERÁ operar por defecto usando exclusivamente servicios y tiers gratuitos.
2. CUANDO una operación pueda generar cargos en AWS, el sistema DEBERÁ advertirlo y requerir confirmación explícita antes de proceder.
3. El sistema NUNCA DEBERÁ crear, modificar o destruir recursos de AWS sin confirmación humana explícita.
4. El sistema DEBERÁ respetar los límites de rate del free tier de Groq y manejar el error 429 con reintento controlado.

### R7 — Calidad, seguridad y distribución

**Historia:** Como mantenedor, quiero que el proyecto sea profesional, testeable y fácil de distribuir, para mantenerlo a largo plazo.

#### Criterios de aceptación

1. La lógica de recolección (collectors/tools) DEBERÁ estar separada del render y ser testeable con clientes mockeados (fake clientset de client-go, mocks del AWS SDK).
2. El sistema NUNCA DEBERÁ hardcodear credenciales, tokens ni API keys; DEBERÁ leerlos de variables de entorno o secretos.
3. CADA cambio de lógica DEBERÁ incluir su test en el mismo cambio.
4. El proyecto DEBERÁ compilar a un binario estático único distribuible vía GoReleaser para linux/darwin en arm64 y amd64.
5. El CI (GitHub Actions) DEBERÁ ejecutar build, test y lint en cada push, sin advertencias.
6. Las credenciales de AWS con las que opere el sistema DEBERÁN ser de solo lectura para las capacidades de la versión 1.
