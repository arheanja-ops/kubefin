# kubefin

> CLI en Go para diagnóstico de EKS y análisis de costos AWS (FinOps), con servidor MCP y agente de IA. Stack 100% gratis.

`kubefin` fusiona el diagnóstico de clústeres Kubernetes/EKS con el análisis de costos de AWS en una sola herramienta. Expone la misma lógica core a través de **tres superficies**: una CLI para humanos, un servidor MCP para clientes LLM (Kiro, Claude Desktop, etc.), y un agente autónomo impulsado por Groq.

Sustituye a los proyectos previos `EKS-resources-review` (Python) y los scripts sueltos de `scripts-sh`.

## Por qué

Las herramientas anteriores hacían shell-out a `kubectl` y parseaban texto (frágil, lento, sin tests). `kubefin` usa las librerías oficiales de Kubernetes (`client-go`) y del SDK de AWS, entrega un binario estático único, y añade una capa de IA opcional que correlaciona uso de recursos con costo real — de pod a factura.

## Principios de diseño

- **Gratis por defecto.** Todo el stack usa tiers gratuitos. Cualquier operación que pueda generar costo en AWS (p. ej. la Cost Explorer API, que cuesta $0.01/request) es **opt-in explícito** y pide confirmación.
- **Determinista primero, IA después.** El diagnóstico duro (métricas, umbrales) es reproducible sin IA. La IA interpreta y explica; nunca reemplaza la lógica.
- **Agent-native.** Cada capacidad es una *tool* atómica reutilizable desde la CLI, MCP o el agente. Un LLM puede lograr lo mismo que un humano.
- **Read-only en v1.** No ejecuta cambios en infraestructura. Lee, correlaciona y recomienda. Cualquier acción de escritura futura requiere confirmación humana.

## Arquitectura de tres superficies

```
                    ┌─────────────────────────────────┐
                    │   internal/tools/  (core logic)  │
                    │   eks · cost · optimize          │
                    └─────────────────────────────────┘
                       ▲            ▲            ▲
          ┌────────────┘            │            └────────────┐
   ┌──────────────┐        ┌────────────────┐        ┌────────────────┐
   │  CLI (cobra) │        │  MCP server    │        │  Agente (Groq) │
   │  kubefin ... │        │  kubefin mcp   │        │  kubefin ask   │
   └──────────────┘        └────────────────┘        └────────────────┘
     humano en terminal      cliente LLM externo       autónomo, tool-use
```

## Stack (todo gratis)

| Capa | Tecnología | Tier gratis |
|---|---|---|
| Lenguaje / CLI | Go 1.27 + cobra | — |
| Diagnóstico EKS | client-go + metrics client | — |
| Datos de costo | Cost & Usage Report (CUR) en S3 | Gratis (storage centavos) |
| Rightsizing | Compute Optimizer | Gratis (recomendaciones básicas) |
| IA / Agente | Groq (Llama 3.3 70B) | 30 req/min, ~14.400 req/día, sin tarjeta |
| Protocolo agentes | MCP (go-sdk oficial) | OSS |
| Repos | GitHub | Gratis |
| CI/CD | GitHub Actions | 2.000 min/mes |
| Release | GoReleaser + GitHub Releases | OSS |
| IaC | Terraform (OSS) + backend S3 | Gratis |
| GitOps (opcional) | Argo CD / Flux | OSS |

> **Nota sobre costos AWS:** la Cost Explorer API cobra $0.01 por request paginado. `kubefin` usa por defecto el **CUR en S3 (gratis)** para datos de costo. La CE API solo se usa con el flag `--use-ce-api` y previo aviso.

## Quickstart

```bash
# Requisitos: Go 1.27+, kubectl configurado, credenciales AWS válidas
git clone <repo> && cd kubefin
make build

# Diagnóstico EKS (usa el contexto actual de kubectl)
./bin/kubefin eks nodes
./bin/kubefin eks report

# Análisis de costos (lee el CUR desde S3, gratis)
./bin/kubefin cost by-service --last 30d

# Agente IA (requiere GROQ_API_KEY)
export GROQ_API_KEY=...
./bin/kubefin ask "¿por qué subió mi factura de EKS este mes?"

# Servidor MCP (para Kiro / Claude Desktop)
./bin/kubefin mcp
```

## Documentación

- [`docs/requirements.md`](docs/requirements.md) — requisitos funcionales y no funcionales
- [`docs/design.md`](docs/design.md) — arquitectura, decisiones técnicas y guardarraíles
- [`docs/tasks.md`](docs/tasks.md) — plan de implementación por fases

## Estado

En diseño. Ver `docs/tasks.md` para el progreso por fases.

## Licencia

Uso personal.
