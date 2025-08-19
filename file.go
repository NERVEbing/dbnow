package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func fileSave(content any, path string) error {
	j, err := json.Marshal(content)
	if err != nil {
		return err
	}
	return os.WriteFile(path, j, 0644)
}

func fileDownload(dir string, link string) error {
	fileName := path.Base(link)
	filePath := filepath.Join(dir, fileName)
	if fileInfo, err := os.Stat(filePath); err == nil {
		if fileInfo.Size() > 1024 {
			return nil
		}
		if err = os.Remove(filePath); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", C.UserAgent)
	client := &http.Client{Timeout: C.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}
	if resp.ContentLength < 1024 {
		return fmt.Errorf("content length is too small: %d", resp.ContentLength)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	_, err = io.Copy(file, resp.Body)

	log.Printf("Downloading file from %s to %s...", link, filePath)

	return err
}

func fileMD5(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(data)), nil
}
