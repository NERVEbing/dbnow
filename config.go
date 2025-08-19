package main

import (
	"log"
	"os"
	"time"
)

const (
	defaultInitialDelay  = time.Second * 10
	defaultInterval      = time.Second * 3600
	defaultPublicURL     = "http://127.0.0.1:8080/"
	defaultDoubanID      = "157489011"
	defaultUserAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
	defaultTimeout       = time.Second * 10
	defaultSaveDir       = "./tmp"
	defaultIndexFileName = "index.json"
)

type Config struct {
	InitialDelay  time.Duration
	Interval      time.Duration
	PublicURL     string
	DoubanID      string
	UserAgent     string
	Timeout       time.Duration
	SaveDir       string
	IndexFileName string
}

var C *Config

func loadConfig() {
	initialDelay, err := time.ParseDuration(getEnv("INITIAL_DELAY", defaultInitialDelay.String()))
	if err != nil {
		log.Fatalf("Error parsing INITIAL_DELAY: %v", err)
	}
	interval, err := time.ParseDuration(getEnv("INTERVAL", defaultInterval.String()))
	if err != nil {
		log.Fatalf("Error parsing INTERVAL: %v", err)
	}
	publicURL := getEnv("PUBLIC_URL", defaultPublicURL)
	doubanID := getEnv("DOUBAN_ID", defaultDoubanID)
	if doubanID == "" {
		log.Fatalf("DOUBAN_ID environment variable is required but not set.")
	}
	userAgent := getEnv("USER_AGENT", defaultUserAgent)
	timeout, err := time.ParseDuration(getEnv("TIMEOUT", defaultTimeout.String()))
	if err != nil {
		log.Fatalf("Error parsing TIMEOUT: %v", err)
	}
	saveDir := getEnv("SAVE_DIR", defaultSaveDir)
	if err = os.MkdirAll(saveDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create directory %s: %v", saveDir, err)
	}
	indexFileName := getEnv("INDEX_FILE_NAME", defaultIndexFileName)

	if C != nil {
		return
	}

	C = &Config{
		InitialDelay:  initialDelay,
		Interval:      interval,
		PublicURL:     publicURL,
		DoubanID:      doubanID,
		UserAgent:     userAgent,
		Timeout:       timeout,
		SaveDir:       saveDir,
		IndexFileName: indexFileName,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
