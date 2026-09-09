package cost

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/arheanja-ops/kubefin/internal/awsx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// Columnas del CUR usadas por el lector. El CUR incluye muchas más; solo se leen
// las necesarias por nombre de cabecera.
const (
	colProductCode = "lineItem/ProductCode"
	colUnblended   = "lineItem/UnblendedCost"
	colCurrency    = "lineItem/CurrencyCode"
	colUsageStart  = "lineItem/UsageStartDate"
)

// ByService lee el CUR y agrega el costo por servicio en los últimos `days` días.
func ByService(ctx context.Context, api awsx.S3API, src Source, days int) ([]model.CostRow, error) {
	return read(ctx, api, src, days, model.GroupByService, "")
}

// ByTag lee el CUR y agrega el costo por el valor del tag `src.TagKey` en los
// últimos `days` días.
func ByTag(ctx context.Context, api awsx.S3API, src Source, days int) ([]model.CostRow, error) {
	if src.TagKey == "" {
		return nil, errors.New("se requiere --tag <clave> para agrupar por tag")
	}
	tagCol := "resourceTags/user:" + src.TagKey
	return read(ctx, api, src, days, model.GroupByTag, tagCol)
}

// read lista los objetos de datos del CUR, los parsea y agrega por la dimensión
// indicada dentro de la ventana [now-days, now).
func read(ctx context.Context, api awsx.S3API, src Source, days int, group model.GroupBy, tagCol string) ([]model.CostRow, error) {
	if src.Bucket == "" {
		return nil, errors.New("se requiere --bucket con el bucket S3 del CUR")
	}

	keys, err := listDataObjects(ctx, api, src)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, ErrNoCUR
	}

	now := time.Now().UTC()
	start := windowStart(now, days)
	period := model.Period{Start: start, End: now}

	agg := map[string]float64{}
	currency := "USD"

	for _, key := range keys {
		cur, err := aggregateObject(ctx, api, src.Bucket, key, group, tagCol, start, now, agg)
		if err != nil {
			return nil, fmt.Errorf("procesando %s: %w", key, err)
		}
		if cur != "" {
			currency = cur
		}
	}

	return toRows(agg, currency, period), nil
}

// listDataObjects lista las claves de objetos de datos del CUR (.csv / .csv.gz)
// bajo el prefijo, excluyendo manifiestos y metadatos.
func listDataObjects(ctx context.Context, api awsx.S3API, src Source) ([]string, error) {
	var keys []string
	var token *string
	for {
		out, err := api.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(src.Bucket),
			Prefix:            aws.String(src.Prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, fmt.Errorf("no se pudieron listar los objetos del CUR: %w", err)
		}
		for _, obj := range out.Contents {
			key := aws.ToString(obj.Key)
			if isCURDataObject(key) {
				keys = append(keys, key)
			}
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	return keys, nil
}

// isCURDataObject filtra los objetos de datos del CUR por extensión.
func isCURDataObject(key string) bool {
	lower := strings.ToLower(key)
	if strings.Contains(lower, "manifest") {
		return false
	}
	return strings.HasSuffix(lower, ".csv") || strings.HasSuffix(lower, ".csv.gz")
}

// aggregateObject descarga y agrega un objeto de datos del CUR dentro de agg.
// Devuelve la divisa detectada.
func aggregateObject(ctx context.Context, api awsx.S3API, bucket, key string,
	group model.GroupBy, tagCol string, start, end time.Time, agg map[string]float64) (string, error) {

	obj, err := api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("no se pudo descargar el objeto: %w", err)
	}
	defer obj.Body.Close()

	var reader io.Reader = obj.Body
	if strings.HasSuffix(strings.ToLower(key), ".gz") {
		gz, gErr := gzip.NewReader(obj.Body)
		if gErr != nil {
			return "", fmt.Errorf("no se pudo abrir el gzip: %w", gErr)
		}
		defer gz.Close()
		reader = gz
	}

	return aggregateCSV(reader, group, tagCol, start, end, agg)
}

// aggregateCSV parsea un CUR en CSV y suma el costo en agg por dimensión.
func aggregateCSV(r io.Reader, group model.GroupBy, tagCol string,
	start, end time.Time, agg map[string]float64) (string, error) {

	cr := csv.NewReader(r)
	cr.ReuseRecord = true
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", nil
		}
		return "", fmt.Errorf("no se pudo leer la cabecera del CUR: %w", err)
	}
	idx := indexColumns(header)

	dimCol := colProductCode
	if group == model.GroupByTag {
		dimCol = tagCol
	}

	currency := ""
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return currency, fmt.Errorf("error leyendo fila del CUR: %w", err)
		}
		if cur := field(rec, idx, colCurrency); cur != "" {
			currency = cur
		}
		if !inWindow(field(rec, idx, colUsageStart), start, end) {
			continue
		}
		cost, perr := strconv.ParseFloat(field(rec, idx, colUnblended), 64)
		if perr != nil || cost == 0 {
			continue
		}
		dim := field(rec, idx, dimCol)
		if dim == "" {
			dim = "(sin valor)"
		}
		agg[dim] += cost
	}
	return currency, nil
}

// indexColumns mapea nombre de columna -> índice.
func indexColumns(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}
	return idx
}

// field devuelve el valor de una columna por nombre, o "" si no existe.
func field(rec []string, idx map[string]int, col string) string {
	if i, ok := idx[col]; ok && i < len(rec) {
		return rec[i]
	}
	return ""
}

// inWindow indica si la marca de tiempo del CUR cae en [start, end).
func inWindow(ts string, start, end time.Time) bool {
	if ts == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// El CUR usa a veces "2006-01-02T15:04:05Z"; RFC3339 lo cubre. Si no
		// parsea, no descartamos la fila para no perder costo.
		return true
	}
	return !t.Before(start) && t.Before(end)
}

// toRows convierte el mapa agregado en filas ordenadas de mayor a menor costo.
func toRows(agg map[string]float64, currency string, period model.Period) []model.CostRow {
	rows := make([]model.CostRow, 0, len(agg))
	for dim, amount := range agg {
		rows = append(rows, model.CostRow{
			Dimension: dim,
			Period:    period,
			Amount:    model.Money{Amount: amount, Currency: currency},
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Amount.Amount > rows[j].Amount.Amount })
	return rows
}
