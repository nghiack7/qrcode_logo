package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/delivery-club/qrlogo"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type qrCode struct {
	Url   string `json:"url"`
	Image string `json:"image"`
}

func main() {
	gracefulTerminateSystem()
	var dir = "app"
	r := gin.Default()
	r.Use(static.Serve("/app/", static.LocalFile(dir, false)))
	r.POST("/create-qrcode", func(ctx *gin.Context) {
		imgBase64 := ctx.Request.FormValue("image")
		url := ctx.Request.FormValue("url")

		imgBase64cleaned := imgBase64[len("data:image/png;base64,"):]

		imgBytes, err := base64.StdEncoding.DecodeString(imgBase64cleaned)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image data"})
			return
		}

		img, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode image data"})
			return
		}

		fname := uuid.New().String()
		currentDir, err := os.Getwd()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current working directory"})
			return
		}

		logoFile := filepath.Join(currentDir, "app/uploaded_logos", fname+".png")
		qrCodeFile := filepath.Join(currentDir, "app/qr_codes", fname+".png")

		imgFile, err := os.Create(logoFile)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create image file"})
			return
		}
		defer imgFile.Close()

		if err := png.Encode(imgFile, img); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode image data"})
			return
		}

		logo, err := loadImage(logoFile)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load logo image"})
			return
		}

		qr, err := qrlogo.Encode(url, logo, 5000, "png")
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
			return
		}

		out, err := os.Create(qrCodeFile)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create QR code file"})
			return
		}
		defer out.Close()

		if _, err := out.Write(qr.Bytes()); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write QR code data"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"fname": fname + ".png"})
	})

	srv := &http.Server{
		Handler:      r,
		Addr:         "localhost:8081",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	currentTime := time.Now()
	fmt.Println("QR Code Generator server is started...", currentTime.Format("Mon 02 Jan 2006 03:04pm"))
	log.Fatal(srv.ListenAndServe())
}

func gracefulTerminateSystem() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r Ctrl + C pressed in terminal")
		os.Exit(0)
	}()
}

func loadImage(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}
