# Diseño — kubefin

## Visión general

`kubefin` es un binario Go único que expone una lógica core (tools) a través de tres superficies: CLI, servidor MCP y agente de IA. El diseño separa estrictamente **recolección de datos** (tools, tipadas y testeables) de **presentación** (render) y de **interpretación** (capa de IA), de modo que la lógica determinista funciona sin IA y las tres superficies comparten el mismo core sin duplicación.

Este documento asume los requisitos de [`requirements.md`](requirements.md).

## Arquitectura de alto nivel

```
┌──────────────────────────────────────────────────────────────┐
│                        Superficies                             │
│   CLI (cobra)        MCP server (go-sdk)      Agente (Groq)     │
│   internal/cli       internal/mcp             internal/agent    │
└───────────┬──────────────────┬───────────────────┬────────────┘
            │                  │                   │
            └──────────────────┴───────────────────┘
                               │  (todas llaman a las mismas tools)
                    ┌──────────▼───────────┐
                    │   internal/tools     │  core, sin dependencias de UI
                    │   eks · cost · optimize
                    └──────────┬───────────┘
             ┌─────────────────┼──────────────────┐
     ┌───────▼──────┐  ┌───────▼───────┐  ┌────────▼────────┐
     │  client-go   │  │  AWS SDK v2   │  │  Groq (ai.Provider)
     │  (K8s API)   │  │  S3/CUR, CO   │  │  OpenAI-compatible
     └──────────────┘  └───────────────┘  └─────────────────┘
```

Render (`internal/render`) es consumido por CLI y agente para formatear salida (tabla/JSON/markdown/confluence). La capa `internal/ai` define la interfaz `Provider` y su implementación Groq.

## Estructura de paquetes

```
kubefin/
├── cmd/kubefin/main.go          # entrypoint, wire de cobra
├── internal/
│   ├── cli/                     # comandos cobra (eks, cost, optimize, ask, mcp)
│   ├── mcp/                     # servidor MCP; registra tools como MCP tools
│   ├── agent/                   # loop de tool-use con Groq
│   ├── ai/                      # Provider interface + impl Groq + redacción
│   ├── tools/
│   │   ├── eks/                 # client-go: nodos, pods, karpenter, requests-vs-uso
│   │   ├── cost/                # CUR reader (S3) + CE API opt-in
│   │   └── optimize/            # Compute Optimizer + puente pod↔node
│   ├── model/                   # structs compartidos (Node, Finding, CostRow, ...)
│   ├── render/                  # tabla/json/markdown/confluence
│   └── awsx/ , k8sx/            # construcción de clientes (config, auth)
├── infra/                       # Terraform: CUR export, bucket S3, IAM read-only
├── .github/workflows/ci.yml     # build + test + lint + release
├── .goreleaser.yaml
├── Dockerfile
├── Makefile
├── go.mod / go.sum
└── docs/                        # requirements.md, design.md, tasks.md
```

Regla estructural: `internal/tools/*` **no importa** `render`, `cli`, `mcp` ni `agent`. El flujo de dependencias apunta hacia el core, nunca al revés. Esto garantiza testeabilidad (R7.1).

## Decisiones técnicas

### D1 — client-go en vez de shell-out a kubectl

**Contexto:** el proyecto Python hacía `subprocess` a `kubectl` y parseaba texto/JSON.
**Decisión:** usar `client-go` con objetos tipados y un `metrics.k8s.io` client para `top`.
**Razón:** elimina el parsing frágil, permite tests con `fake.NewSimpleClientset`, y es el ecosistema nativo de K8s (todo escrito en Go).
**Consecuencia:** autenticación EKS vía kubeconfig del contexto actual; no se reimplementa el resolver de AWS-IAM auth, se reutiliza el del kubeconfig.

### D2 — CUR-first, Cost Explorer API opt-in

**Contexto:** la CE API cobra $0.01 por request paginado (confirmado en AWS FAQ oficial). Un agente iterativo podría disparar el costo.
**Decisión:** por defecto leer el **Cost & Usage Report** desde S3 (Parquet/CSV), gratis salvo storage. La CE API solo se activa con `--use-ce-api` y aviso de costo.
**Razón:** cumple el requisito de operación gratuita (R2, R6).
**Consecuencia:** el CUR tarda ~24h en poblarse y requiere configuración previa (creada por Terraform en `infra/`). Para cuentas nuevas sin CUR, el sistema explica cómo configurarlo (R2.5).

### D3 — Una tool, tres superficies (agent-native)

**Decisión:** cada capacidad se implementa una sola vez en `internal/tools/*` como función con entrada/salida tipada. CLI, MCP y agente son adaptadores delgados.
**Razón:** evita duplicación y garantiza que un LLM (vía MCP o agente) logre lo mismo que un humano (principio agent-native).
**Consecuencia:** las firmas de las tools deben ser serializables (para MCP schema y para el contexto del agente).

### D4 — Groq como Provider por defecto, interfaz intercambiable

**Decisión:** `internal/ai` define `type Provider interface { Chat(ctx, messages, tools) (Response, error) }`. Groq se implementa sobre su API OpenAI-compatible.
**Razón:** Groq tiene free tier real (30 req/min, ~14.400 req/día, sin tarjeta) y es OpenAI-compatible, así que cambiar a OpenAI/Ollama es trivial.
**Consecuencia:** manejo explícito de rate limit 429 con backoff (R6.4). El modelo por defecto es Llama 3.3 70B; configurable por env.

### D5 — Salida estructurada, no prosa

**Decisión:** las tools y el agente devuelven structs (`model.Finding`, `model.CostRow`), y el render los pinta. El agente usa tool-use / JSON mode para producir `[]Finding`.
**Razón:** consistencia entre superficies, testeabilidad, y evita "alucinación de tablas".
**Consecuencia:** el prompt del agente instruye salida estructurada; se valida el JSON antes de renderizar.

### D6 — Read-only y guardarraíles de costo

**Decisión:** v1 es estrictamente read-only. Ninguna tool muta infraestructura. Operaciones con costo (CE API) o creación de recursos (Terraform) requieren confirmación humana explícita.
**Razón:** requisitos R5.4, R6.2, R6.3 y las reglas de seguridad del proyecto.
**Consecuencia:** los comandos que sugieren acciones (p. ej. aplicar rightsizing) solo imprimen el comando; no lo ejecutan.

## Modelo de datos (núcleo)

```go
// model/finding.go
type Severity string // "info" | "warning" | "critical"

type Finding struct {
    Severity        Severity
    Title           string
    Evidence        string
    Recommendation  string
    Command         string   // sugerido, nunca ejecutado automáticamente
    EstimatedSaving *Money   // nil si no aplica
}

// model/cost.go
type CostRow struct {
    Dimension string    // servicio, tag, cuenta
    Period    Period
    Amount    Money
}
```

## Seguridad

- `GROQ_API_KEY` y credenciales AWS se leen de entorno/secretos; nunca se hardcodean ni se loguean (R7.2).
- Credenciales AWS de solo lectura para las capacidades v1 (R7.6); el rol IAM lo define Terraform con las mínimas acciones necesarias (`ce:Get*` solo si opt-in, `s3:GetObject` sobre el bucket CUR, `compute-optimizer:Get*`).
- Redacción opcional de account IDs, ARNs e IPs internas antes de enviar contexto a un proveedor de IA remoto (R5.5).
- Todo contenido externo (salida de APIs, respuestas del LLM) se trata como no confiable; las sugerencias del LLM nunca se ejecutan sin humano.

## Testing

- Tools EKS: `fake.NewSimpleClientset` de client-go con objetos K8s sembrados.
- Tools cost/optimize: interfaces sobre el AWS SDK v2 mockeadas; fixtures de CUR en Parquet/CSV pequeños.
- Agente: Provider mock que devuelve secuencias de tool-use deterministas.
- Tests de tabla; se verifica que rompan al romper la lógica (R7.3).

## Distribución y CI/CD

- **GoReleaser**: binarios linux/darwin arm64+amd64 + GitHub Releases; opcional fórmula Homebrew.
- **GitHub Actions**: `build → test → golangci-lint → release` en tags. Zero warnings.
- **Dockerfile** multi-stage para el servidor MCP.
- **Terraform** (`infra/`): CUR export, bucket S3, rol IAM read-only. `terraform apply` requiere confirmación (crea recursos AWS).
- **GitOps (opcional)**: Argo CD/Flux sincroniza el manifiesto del MCP server si se despliega en clúster; no requerido para uso CLI local.
