package main

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxUpload = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body,  maxUpload)
	videoIDStr := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "jwt token not in header", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "user not validated", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "get video failed", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "video user is not the same", err)
		return
	}
	

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "file not formed", err)
		return
	}
	defer file.Close()
	mediaType := header.Header.Get("Content-Type")
	result, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "mime not parse media type", err)
		return
	}
	if result != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "mime not correct media type", err)
		return
	}

	assetPath := getAssetPath(mediaType)

	tempFile, err := os.CreateTemp("", "tubley-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err = io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to copy to temp file", err)
		return
	}

	if _, err = tempFile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to reset to temp file", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	keyPrefix := "other"
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get video aspect ratio", err)
		return
	}
	if aspectRatio == "16:9" {
		keyPrefix = "landscape"
	} else if aspectRatio == "9:16" {
		keyPrefix = "portrait"
	} 


	assetPath = filepath.Join(keyPrefix, assetPath)
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
	Bucket: aws.String(cfg.s3Bucket),
	Key: aws.String(assetPath),
	Body: tempFile,
	ContentType: aws.String(mediaType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading file to S3", err)
		return
	}

	videoUrl := cfg.getObjectUrl(assetPath)
	video.VideoURL = &videoUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
