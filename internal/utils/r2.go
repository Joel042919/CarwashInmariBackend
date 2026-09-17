package utils

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

func NewR2Client(ctx context.Context) (*R2Client, error) {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("R2_BUCKET_NAME")
	publicURL := os.Getenv("R2_PUBLIC_CUSTOM_DOMAIN")
	if publicURL == "" {
		publicURL = os.Getenv("R2_PUBLIC_URL")
	}
	if publicURL == "" && accountID != "" && bucketName != "" {
		publicURL = fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s", accountID, bucketName)
	}
	for len(publicURL) > 0 && publicURL[len(publicURL)-1] == '/' {
		publicURL = publicURL[:len(publicURL)-1]
	}

	if accountID == "" || accessKey == "" || secretKey == "" || bucketName == "" {
		return nil, nil
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     "auto",
				HostnameImmutable: true,
			}, nil
		},
	)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	return &R2Client{
		client:     s3.NewFromConfig(cfg),
		bucketName: bucketName,
		publicURL:  publicURL,
	}, nil
}

func (r *R2Client) SubirArchivo(ctx context.Context, file io.Reader, key, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	// Devuelve la URL pública accesible para React Native
	urlFinal := fmt.Sprintf("%s/%s", r.publicURL, key)
	return urlFinal, nil
}

func (r *R2Client) ObtenerArchivo(ctx context.Context, key string) (io.ReadCloser, string, error) {
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	contentType := "application/octet-stream"
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return out.Body, contentType, nil
}
