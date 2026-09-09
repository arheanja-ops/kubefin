// Package cost implementa el análisis de costos: lector del Cost & Usage Report
// (CUR) desde S3 (gratis por defecto) y una ruta opt-in a la Cost Explorer API.
package cost

import (
	"errors"
	"time"
)

// Source describe dónde vive el CUR en S3.
type Source struct {
	// Bucket es el bucket S3 donde se exporta el CUR.
	Bucket string
	// Prefix es el prefijo/carpeta del export dentro del bucket.
	Prefix string
	// TagKey es la clave del tag de asignación de costos a usar cuando se agrupa
	// por tag (columna resourceTags/user:<TagKey> en el CUR).
	TagKey string
}

// ErrNoCUR indica que no se encontró ningún dato de CUR bajo la fuente dada
// (típico en cuentas nuevas sin export configurado). Las capas superiores lo
// detectan para mostrar la guía de configuración (R2.5).
var ErrNoCUR = errors.New("no se encontró ningún dato de CUR en la fuente indicada")

// NoCURGuidance es el mensaje que explica cómo habilitar el CUR cuando no existe
// (R2.5). No falla en silencio: se muestra al usuario.
const NoCURGuidance = `No hay datos de CUR disponibles todavía.

El Cost & Usage Report (CUR) es la fuente de costo gratuita de kubefin. Para habilitarlo:

  1. Aplica la infraestructura de infra/ (Terraform): crea el bucket S3 y el
     export del CUR con ` + "`terraform apply`" + ` (crea recursos AWS; requiere confirmación).
  2. AWS tarda hasta ~24h en poblar el primer reporte tras crearlo.
  3. Reintenta con --bucket <bucket> --prefix <prefijo> apuntando al export.

Alternativa (con costo): usa --use-ce-api para consultar la Cost Explorer API
($0.01 por request). No es gratis; se pedirá confirmación.`

// windowStart devuelve el inicio de una ventana de los últimos n días desde now.
func windowStart(now time.Time, days int) time.Time {
	return now.AddDate(0, 0, -days)
}
