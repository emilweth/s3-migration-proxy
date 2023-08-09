package main

import (
	"flag"
	"log"
	"time"

	kvstore "github.com/emilweth/s3-migration-proxy/internal/cache"
	"github.com/emilweth/s3-migration-proxy/internal/config"
	"github.com/emilweth/s3-migration-proxy/internal/s3client"
	"github.com/emilweth/s3-migration-proxy/internal/s3proxy"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to the config.yaml file")
	flag.Parse()

	// Load configuration from the provided file path
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize the source S3 client using the source S3 configuration
	sourceClient, err := s3client.NewS3Client(cfg.S3.Source)
	if err != nil {
		log.Fatalf("Failed to create source S3 client: %v", err)
	}

	// Initialize the target S3 client using the target S3 configuration
	targetClient, err := s3client.NewS3Client(cfg.S3.Target)
	if err != nil {
		log.Fatalf("Failed to create target S3 client: %v", err)
	}

	cacheDuration := time.Duration(cfg.S3.CacheErrorDuration) * time.Second

	store := kvstore.New()
	go store.Cleanup(3 * time.Minute)

	// Start the S3 proxy server
	proxy := s3proxy.NewS3Proxy(sourceClient, targetClient, cfg.S3.Source.BucketName, cfg.S3.Target.BucketName, cfg.HTTP, cacheDuration)
	proxy.Start()
}
