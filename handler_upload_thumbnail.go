package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediatype, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media type", err)
		return
	}
	if mediatype != "image/jpeg" && mediatype != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Uploaded file type is not image type", nil)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting video from database", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the video owner", nil)
		return
	}

	fileExtension := strings.Split(mediatype, "/")[1]

	// random ID generation
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		panic("Error generating random bytes")
	}
	id := base64.RawURLEncoding.EncodeToString(key)

	fileName := fmt.Sprintf("%v.%v", id, fileExtension)
	filePath := filepath.Join(cfg.assetsRoot, fileName)

	thumbnailFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating thumbnail file", err)
		return
	}

	if _, err = io.Copy(thumbnailFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error copying thumnail image", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, fileName)
	video.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
