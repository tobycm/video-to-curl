package routes

import (
	"context"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qeesung/image2ascii/convert"
	"github.com/wader/goutubedl"

	"vsus.app/tobycm/video-to-curl/utils"
)

type WatchRouteOptions struct {
	TempDir string
}

var asciiOptions = convert.DefaultOptions

func AddWatchRoute(router *gin.RouterGroup, options WatchRouteOptions) {

	if _, err := os.Stat(options.TempDir); os.IsNotExist(err) {
		if err := os.Mkdir(options.TempDir, 0777); err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat(options.TempDir + "/uploads"); os.IsNotExist(err) {
		if err := os.Mkdir(options.TempDir+"/uploads", 0777); err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat(options.TempDir + "/youtube"); os.IsNotExist(err) {
		if err := os.Mkdir(options.TempDir+"/youtube", 0777); err != nil {
			panic(err)
		}
	}

	asciiOptions.FixedWidth = 160
	asciiOptions.FixedHeight = 45

	router.Use(func(c *gin.Context) {
		c.Writer.Write([]byte("Initializing...\n"))
		c.Writer.Flush()

		sWidth := c.Query("width")
		sHeight := c.Query("height")

		if sWidth == "" {
			asciiOptions.FixedWidth = 160
		}

		if sHeight == "" {
			asciiOptions.FixedHeight = 45
		}

		if match, _ := regexp.MatchString("^[0-9]*$", sWidth); !match || sWidth == "0" {
			c.JSON(400, gin.H{
				"message": "Invalid width value",
			})
			c.Abort()
			return
		}

		parsedWidth, err := strconv.Atoi(sWidth)
		if err != nil || parsedWidth > 2000 {
			c.JSON(400, gin.H{
				"message": "Invalid width value",
			})
			c.Abort()
			return
		}
		asciiOptions.FixedWidth = parsedWidth

		if match, _ := regexp.MatchString("^[0-9]*$", sHeight); !match || sHeight == "0" {
			c.JSON(400, gin.H{
				"message": "Invalid height value",
			})
			c.Abort()
			return
		}

		parsedHeight, err := strconv.Atoi(sHeight)
		if err != nil || parsedHeight > 2000 {
			c.JSON(400, gin.H{
				"message": "Invalid height value",
			})
			c.Abort()
			return
		}

		asciiOptions.FixedHeight = parsedHeight

	})

	router.POST("/upload/:name", func(c *gin.Context) {
		name := c.Param("name")

		match, _ := regexp.MatchString("^[a-zA-Z0-9_-]*$", name)

		if !match {
			name = utils.RandomString(16)
		} else {
			if len(name) > 20 {
				name = name[:20]
			}
		}

		filename := options.TempDir + "/uploads/" + name + ".mp4"

		file, err := os.Create(filename)
		if err != nil {
			c.JSON(500, gin.H{
				"message": "Failed to create file",
			})
			return
		}
		defer file.Close()

		c.Writer.Write([]byte("Uploading...\n"))
		c.Writer.Flush()

		if _, err := io.Copy(file, c.Request.Body); err != nil {
			c.JSON(500, gin.H{
				"message": "Failed to write to file",
			})
			return
		}

		c.Writer.Write([]byte("Processing...\n"))
		c.Writer.Flush()

		utils.ServeVideo(c, filename, &asciiOptions)
	})

	router.GET("/youtube/:id", func(c *gin.Context) {
		id := c.Param("id")
		match, _ := regexp.MatchString("^[a-zA-Z0-9_-]*$", id)

		if len(id) > 11 || !match {
			c.JSON(400, gin.H{
				"message": "Invalid video ID",
			})
			return
		}

		result, err := goutubedl.New(context.Background(), "https://www.youtube.com/watch?v="+id, goutubedl.Options{})
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"message": "Failed to fetch video",
			})
			return
		}

		c.Writer.Write([]byte("Downloading...\n"))
		c.Writer.Flush()

		downloadResult, err := result.Download(context.Background(), "best[height<=720]")
		if err != nil {
			log.Fatal(err)
		}
		defer downloadResult.Close()

		filename := options.TempDir + "/youtube/" + id + ".mp4"

		f, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"message": "Failed to create file",
			})
			return
		}
		defer f.Close()
		io.Copy(f, downloadResult)

		c.Writer.Write([]byte("Processing...\n"))

		utils.ServeVideo(c, filename, &asciiOptions)
	})

}
