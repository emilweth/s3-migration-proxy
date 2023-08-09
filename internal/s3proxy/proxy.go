package s3proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"

	"github.com/emilweth/s3-migration-proxy/internal/cache"

	"github.com/emilweth/s3-migration-proxy/internal/config"
	"github.com/emilweth/s3-migration-proxy/internal/s3client"
)

type S3Proxy struct {
	sourceClient     *s3client.Client
	targetClient     *s3client.Client
	sourceBucketName string
	targetBucketName string
	httpConfig       config.HTTPConfig
	cache            *kvstore.Store
	cacheDuration    time.Duration
}

func NewS3Proxy(source, target *s3client.Client, sourceBucketName, targetBucketName string, httpCfg config.HTTPConfig, cacheDuration time.Duration) *S3Proxy {
	return &S3Proxy{
		sourceClient:     source,
		targetClient:     target,
		sourceBucketName: sourceBucketName,
		targetBucketName: targetBucketName,
		httpConfig:       httpCfg,
		cache:            kvstore.New(),
		cacheDuration:    cacheDuration,
	}
}

func (p *S3Proxy) Start() {
	http.HandleFunc("/", p.handleRequest)
	address := fmt.Sprintf(":%d", p.httpConfig.Port)
	log.Printf("Starting S3 proxy server on %s", address)
	http.ListenAndServe(address, nil)
}

func (p *S3Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	objectKey := r.URL.Path[1:] // Remove the leading '/'

	// Check the cache first
	_, found := p.cache.Get(objectKey)
	if found {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var result *s3.GetObjectOutput
	var err error
	var objectSource string

	// Try fetching the object from the target S3 bucket.
	result, err = p.targetClient.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(p.targetBucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		// Object not in target, try fetching from source bucket.
		result, err = p.sourceClient.S3.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(p.sourceBucketName),
			Key:    aws.String(objectKey),
		})

		if err != nil {
			w.Header().Set("Cache-Control", "public, max-age=300")
			log.Errorf("Failed to fetch object: %v", err)
			http.Error(w, "Failed to fetch object", http.StatusInternalServerError)

			// Add the objectKey to cache
			p.cache.Set(objectKey, "NoSuchKey", p.cacheDuration)
			return
		}

		objectSource = "source"

		// Use a Goroutine to copy the object in the background.
		go func() {
			bodyBytes, readErr := ioutil.ReadAll(result.Body)
			if readErr != nil {
				log.WithFields(log.Fields{
					"object_key": objectKey,
				}).Errorf("Failed to read object body: %v", readErr)
				return
			}

			// Use the buffer as input for the PutObject call.
			_, copyErr := p.targetClient.S3.PutObject(&s3.PutObjectInput{
				Bucket: aws.String(p.targetBucketName),
				Key:    aws.String(objectKey),
				Body:   bytes.NewReader(bodyBytes),
			})

			if copyErr != nil {
				log.WithFields(log.Fields{
					"object_key": objectKey,
					"source":     p.sourceBucketName,
					"target":     p.targetBucketName,
				}).Error("Failed to copy object from source to target")
			} else {
				log.WithFields(log.Fields{
					"object_key": objectKey,
					"source":     p.sourceBucketName,
					"target":     p.targetBucketName,
				}).Info("Object migrated from source to target")
			}
		}()
	} else {
		objectSource = "target"
	}

	// Serve the object.
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	for name, valuePtr := range result.Metadata {
		if valuePtr != nil {
			w.Header().Add(name, *valuePtr)
		}
	}
	io.Copy(w, result.Body)

	// Logging at the end with object source information.
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.WithFields(log.Fields{
		"client":        host,
		"object_key":    objectKey,
		"object_source": objectSource,
	}).Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
}
