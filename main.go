package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	loadConfig()
	schedule()
}

func schedule() {
	log.Printf("Running schedule after %s...", C.InitialDelay)
	<-time.After(C.InitialDelay)
	run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(C.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			run()
		case <-quit:
			log.Println("Received quit signal, shutting down.")
			return
		}
	}
}

func run() {
	log.Println("Fetching Douban RSS feed...")
	items, err := doubanFetch()
	if err != nil {
		log.Fatalf("Failed to fetch Douban RSS feed: %v", err)
	}

	log.Println("Saving Douban RSS items to a local file...")
	if err = doubanSave(items); err != nil {
		log.Fatalf("Failed to save local file: %v", err)
	}

	log.Println("Cleaning up local file...")
	if err = doubanCleanup(); err != nil {
		log.Fatalf("Failed to clean up local file: %v", err)
	}

	log.Println("Done!")
}
