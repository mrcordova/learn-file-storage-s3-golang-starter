package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(mediaType string) string {
	base := make([]byte, 32)
	_, err := rand.Read(base)
	if err != nil {
		panic("failed to generate random bytes")
	}
	id := base64.RawURLEncoding.EncodeToString(base)

	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", id, ext)
}
func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}
func (cfg apiConfig) getObjectUrl(key string) string  {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region,key)
}
func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getVideoAspectRatio(filePath string) (string, error)  {
	cmd := exec.Command("ffprobe", "-v","error", "-print_format","json", "-show_streams", filePath)

	var out bytes.Buffer

	type Stream struct {
		Width int `json:"width"`
		Height int `json:"height"`
	}
	type Response struct {
		Streams []Stream `json:"streams"`
	}

	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	

	response := Response{}
	response.Streams = make([]Stream, 0)
	err = json.Unmarshal(out.Bytes(), &response)
	if err != nil {
		return "", err
	}
	
	aspectRatio := response.Streams[0].Width / response.Streams[0].Height
	aspectRatioStr := ""
	if aspectRatio == 0 {
		aspectRatioStr = "9:16"
	} else if aspectRatio == 1 {
		aspectRatioStr = "16:9"
	} else {
		aspectRatioStr = "other"
	}
	return aspectRatioStr, nil
}
