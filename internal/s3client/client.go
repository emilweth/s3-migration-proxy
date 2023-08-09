package s3client

import (
	"crypto/tls"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/emilweth/s3-migration-proxy/internal/config"
)

type Client struct {
	S3 *s3.S3
}

func NewS3Client(cfg config.S3Config) (*Client, error) {
	awsConfig := &aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Endpoint:    aws.String(cfg.Endpoint),
	}

	if cfg.Protocol == "https" {
		awsConfig.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Depending on your needs, you might want to disable this line
			},
		})
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	client := &Client{
		S3: s3.New(sess),
	}

	return client, nil
}
