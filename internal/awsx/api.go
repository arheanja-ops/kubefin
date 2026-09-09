// Package awsx construye los clientes de AWS (config, credenciales, S3, Compute Optimizer).
//
// Las operaciones de AWS que consumen las tools se exponen como interfaces
// pequeñas (S3API, ComputeOptimizerAPI) para poder mockearlas en tests sin
// tocar la red (R7.1).
package awsx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3API es el subconjunto de operaciones de S3 que usa el lector de CUR.
type S3API interface {
	ListObjectsV2(ctx context.Context, in *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, in *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// ComputeOptimizerAPI es el subconjunto de operaciones de Compute Optimizer que
// usa la tool de rightsizing.
type ComputeOptimizerAPI interface {
	GetEC2InstanceRecommendations(ctx context.Context, in *computeoptimizer.GetEC2InstanceRecommendationsInput, optFns ...func(*computeoptimizer.Options)) (*computeoptimizer.GetEC2InstanceRecommendationsOutput, error)
	GetEBSVolumeRecommendations(ctx context.Context, in *computeoptimizer.GetEBSVolumeRecommendationsInput, optFns ...func(*computeoptimizer.Options)) (*computeoptimizer.GetEBSVolumeRecommendationsOutput, error)
}
