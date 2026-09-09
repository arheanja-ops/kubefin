package cost

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// fakeS3 implementa awsx.S3API con objetos en memoria.
type fakeS3 struct {
	objects map[string][]byte
}

func (f *fakeS3) ListObjectsV2(_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	var contents []s3types.Object
	prefix := aws.ToString(in.Prefix)
	for key := range f.objects {
		if strings.HasPrefix(key, prefix) {
			contents = append(contents, s3types.Object{Key: aws.String(key)})
		}
	}
	return &s3.ListObjectsV2Output{Contents: contents, IsTruncated: aws.Bool(false)}, nil
}

func (f *fakeS3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data, ok := f.objects[aws.ToString(in.Key)]
	if !ok {
		return nil, errors.New("NoSuchKey")
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

// curCSV genera un CUR CSV mínimo con las columnas relevantes.
func curCSV(rows [][3]string, tagCol string) string {
	var b strings.Builder
	header := colProductCode + "," + colUnblended + "," + colCurrency + "," + colUsageStart
	if tagCol != "" {
		header += "," + tagCol
	}
	b.WriteString(header + "\n")
	for _, r := range rows {
		line := r[0] + "," + r[1] + ",USD," + r[2]
		if tagCol != "" {
			line += ",team-a"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func gzipBytes(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(s)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestByService_AggregatesCSV(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	csv := curCSV([][3]string{
		{"AmazonEC2", "10.50", now},
		{"AmazonEC2", "4.50", now},
		{"AmazonS3", "2.00", now},
	}, "")
	api := &fakeS3{objects: map[string][]byte{"cur/data-1.csv": []byte(csv)}}

	rows, err := ByService(context.Background(), api, Source{Bucket: "b", Prefix: "cur"}, 30)
	if err != nil {
		t.Fatalf("ByService error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("esperaba 2 servicios, obtuve %d", len(rows))
	}
	// Ordenado de mayor a menor: EC2 (15.00) antes que S3 (2.00).
	if rows[0].Dimension != "AmazonEC2" || rows[0].Amount.Amount != 15.0 {
		t.Errorf("EC2 esperado 15.00, obtuve %s=%.2f", rows[0].Dimension, rows[0].Amount.Amount)
	}
	if rows[1].Dimension != "AmazonS3" {
		t.Errorf("segundo servicio esperado AmazonS3, obtuve %s", rows[1].Dimension)
	}
}

func TestByService_ReadsGzip(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	csv := curCSV([][3]string{{"AmazonRDS", "7.25", now}}, "")
	api := &fakeS3{objects: map[string][]byte{"cur/data.csv.gz": gzipBytes(t, csv)}}

	rows, err := ByService(context.Background(), api, Source{Bucket: "b", Prefix: "cur"}, 30)
	if err != nil {
		t.Fatalf("ByService(gzip) error: %v", err)
	}
	if len(rows) != 1 || rows[0].Amount.Amount != 7.25 {
		t.Fatalf("esperaba RDS=7.25, obtuve %+v", rows)
	}
}

func TestByService_WindowFilters(t *testing.T) {
	old := time.Now().UTC().AddDate(0, 0, -90).Format(time.RFC3339)
	recent := time.Now().UTC().Format(time.RFC3339)
	csv := curCSV([][3]string{
		{"AmazonEC2", "100.00", old},
		{"AmazonEC2", "5.00", recent},
	}, "")
	api := &fakeS3{objects: map[string][]byte{"cur/d.csv": []byte(csv)}}

	rows, err := ByService(context.Background(), api, Source{Bucket: "b", Prefix: "cur"}, 30)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(rows) != 1 || rows[0].Amount.Amount != 5.0 {
		t.Fatalf("la ventana de 30d debería excluir la fila de hace 90d; obtuve %+v", rows)
	}
}

func TestByTag_GroupsByTagValue(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	tagCol := "resourceTags/user:Team"
	csv := curCSV([][3]string{{"AmazonEC2", "12.00", now}}, tagCol)
	api := &fakeS3{objects: map[string][]byte{"cur/d.csv": []byte(csv)}}

	rows, err := ByTag(context.Background(), api, Source{Bucket: "b", Prefix: "cur", TagKey: "Team"}, 30)
	if err != nil {
		t.Fatalf("ByTag error: %v", err)
	}
	if len(rows) != 1 || rows[0].Dimension != "team-a" {
		t.Fatalf("esperaba dimensión team-a, obtuve %+v", rows)
	}
}

func TestByService_NoCUR(t *testing.T) {
	api := &fakeS3{objects: map[string][]byte{"cur/manifest.json": []byte("{}")}}
	_, err := ByService(context.Background(), api, Source{Bucket: "b", Prefix: "cur"}, 30)
	if !errors.Is(err, ErrNoCUR) {
		t.Fatalf("esperaba ErrNoCUR cuando solo hay manifiestos, obtuve %v", err)
	}
}

func TestByTag_RequiresTagKey(t *testing.T) {
	api := &fakeS3{objects: map[string][]byte{}}
	_, err := ByTag(context.Background(), api, Source{Bucket: "b"}, 30)
	if err == nil {
		t.Fatal("esperaba error cuando falta TagKey")
	}
}
