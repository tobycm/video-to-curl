package routes

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wader/goutubedl"

	"vsus.app/tobycm/video-to-curl/utils"
)

type WatchRouteOptions struct {
	TempDir string
}

func parseAsciiOptions(c *gin.Context) (utils.AsciiOptions, error) {
	c.Writer.Write([]byte("Initializing...\n"))
	c.Writer.Flush()

	asciiOptions := utils.AsciiOptions{
		Width:  160,
		Height: 0,
	}

	sWidth := c.Query("width")
	sHeight := c.Query("height")

	if sWidth != "" {
		if match, _ := regexp.MatchString("^[0-9]*$", sWidth); !match || sWidth == "0" {
			c.JSON(400, gin.H{
				"message": "Invalid width value",
			})
			c.Abort()
			return asciiOptions, errors.New("invalid width value")
		}

		parsedWidth, err := strconv.Atoi(sWidth)
		if err != nil || parsedWidth > 2000 {
			c.JSON(400, gin.H{
				"message": "Invalid width value",
			})
			c.Abort()
			return asciiOptions, errors.New("invalid width value")
		}
		asciiOptions.Width = parsedWidth
	}

	if sHeight != "" {
		if match, _ := regexp.MatchString("^[0-9]*$", sHeight); !match || sHeight == "0" {
			c.JSON(400, gin.H{
				"message": "Invalid height value",
			})
			c.Abort()
			return asciiOptions, errors.New("invalid height value")
		}

		parsedHeight, err := strconv.Atoi(sHeight)
		if err != nil || parsedHeight > 2000 {
			c.JSON(400, gin.H{
				"message": "Invalid height value",
			})
			c.Abort()
			return asciiOptions, errors.New("invalid height value")
		}

		asciiOptions.Height = parsedHeight

	}

	return asciiOptions, nil
}

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

	router.Use()

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

		asciiOptions, err := parseAsciiOptions(c)
		if err != nil {
			return
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

		withSubtitles := c.Query("sub")
		match, _ = regexp.MatchString("^[a-zA-Z0-9_-]*$", withSubtitles)
		if !match {
			withSubtitles = ""
		}

		asciiOptions, err := parseAsciiOptions(c)
		if err != nil {
			return
		}

		result, err := goutubedl.New(context.Background(), "https://www.youtube.com/watch?v="+id, goutubedl.Options{DownloadSubtitles: true})
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"message": "Failed to fetch video",
			})
			return
		}

		subLangs := make([]string, len(result.Info.Subtitles))

		if withSubtitles != "" {
			c.Writer.Write([]byte("Available subtitles:\n"))

			i := 0
			for lang := range result.Info.Subtitles {
				subLangs[i] = lang
				i++
			}

			c.Writer.Write([]byte(strings.Join(subLangs, ", ") + "\n"))
			c.Writer.Flush()

			time.Sleep(3 * time.Second)
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

		subtitleFilename := ""

		if withSubtitles != "" && result.Info.Subtitles[withSubtitles] != nil {
			var subtitle goutubedl.Subtitle

			for _, s := range result.Info.Subtitles[withSubtitles] {
				if s.Ext == "vtt" {
					subtitle = s
					break
				}
			}

			if subtitle.URL == "" {
				c.Writer.Write([]byte("No subtitles for " + withSubtitles + " found\n"))
				c.Writer.Flush()

				time.Sleep(3 * time.Second)
			} else {
				subtitleFilename = options.TempDir + "/youtube/" + id + "-" + subtitle.Language + "." + subtitle.Ext
				subtitleFile, err := os.Create(subtitleFilename)
				if err != nil {
					log.Fatal(err)
					c.JSON(500, gin.H{
						"message": "Failed to create subtitle file",
					})
					return
				}
				defer subtitleFile.Close()

				response, err := http.Get(subtitle.URL)
				if err != nil {
					log.Fatal(err)
					c.JSON(500, gin.H{
						"message": "Failed to download subtitle",
					})
					return
				}
				defer response.Body.Close()

				_, err = io.Copy(subtitleFile, response.Body)
				if err != nil {
					log.Fatal(err)
					c.JSON(500, gin.H{
						"message": "Failed to write subtitle file",
					})
					return
				}
			}
		}

		c.Writer.Write([]byte("Processing...\n"))

		utils.ServeVideoWithSubtitle(c, filename, subtitleFilename, &asciiOptions)
	})

}
