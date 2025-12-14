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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	imageType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if imageType != "image/png" && imageType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "bad file type", err)
		return

	}
	imageFileExtension := strings.Split(imageType, "/")[1]
	defer file.Close()
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video", err)
		return
	}
	var randBytes []byte
	rand.Read(randBytes)
	name := base64.RawURLEncoding.Strict().EncodeToString(randBytes)
	path := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%v.%v", name, imageFileExtension))
	f, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create thumbnail file", err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	newURL := fmt.Sprintf("http://localhost:%v/assets/%v.%v", 8091, name, imageFileExtension)
	video.ThumbnailURL = &newURL

	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
