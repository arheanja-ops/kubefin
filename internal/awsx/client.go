package awsx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Identity resume la identidad AWS efectiva de la sesión.
type Identity struct {
	Account string
	ARN     string
	UserID  string
}

// Clients agrupa la config y los clientes de AWS que consumen las tools de costo
// y optimización. Se construyen a partir de la cadena de credenciales estándar
// (AWS_PROFILE / SSO / variables de entorno); nunca se hardcodean credenciales.
type Clients struct {
	Config           aws.Config
	S3               S3API
	ComputeOptimizer ComputeOptimizerAPI
	sts              stsAPI
}

// stsAPI abstrae GetCallerIdentity para poder mockear la verificación en tests.
type stsAPI interface {
	GetCallerIdentity(ctx context.Context, in *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// Load carga la configuración de AWS de la región indicada (o la del entorno si
// region == "") y construye los clientes. No verifica credenciales aquí; usa
// VerifyIdentity para ello.
func Load(ctx context.Context, region string) (*Clients, error) {
	opts := []func(*config.LoadOptions) error{}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar la configuración de AWS: %w", err)
	}
	return &Clients{
		Config:           cfg,
		S3:               s3.NewFromConfig(cfg),
		ComputeOptimizer: computeoptimizer.NewFromConfig(cfg),
		sts:              sts.NewFromConfig(cfg),
	}, nil
}

// VerifyIdentity confirma que las credenciales son válidas devolviendo la
// identidad efectiva. Falla con un mensaje claro si las credenciales faltan o
// están expiradas (guía al usuario a renovar la sesión SSO).
func (c *Clients) VerifyIdentity(ctx context.Context) (*Identity, error) {
	out, err := c.sts.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("no se pudo verificar la identidad de AWS (¿credenciales expiradas? prueba `aws sso login`): %w", err)
	}
	return &Identity{
		Account: aws.ToString(out.Account),
		ARN:     aws.ToString(out.Arn),
		UserID:  aws.ToString(out.UserId),
	}, nil
}

// CostExplorer construye el cliente de Cost Explorer. Se crea bajo demanda
// porque la CE API tiene costo ($0.01/request) y solo se usa con opt-in.
func (c *Clients) CostExplorer() *costexplorer.Client {
	return costexplorer.NewFromConfig(c.Config)
}
