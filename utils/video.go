package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qeesung/image2ascii/convert"
)

func ServeVideoWithSubtitle(c *gin.Context, filename string, subtitleFile string, asciiOptions *convert.Options) {
	serveVideo(c, filename, subtitleFile, asciiOptions)
}

func ServeVideo(c *gin.Context, filename string, asciiOptions *convert.Options) {
	serveVideo(c, filename, "", asciiOptions)
}

func serveVideo(c *gin.Context, filename string, subtitleFile string, asciiOptions *convert.Options) {
	var cmd *exec.Cmd

	if subtitleFile != "" {
		cmd = exec.Command("ffmpeg", "-an", "-readrate", "1", "-i", filename, "-vf", "subtitles="+subtitleFile, "-f", "image2pipe", "pipe:1")
	} else {
		cmd = exec.Command("ffmpeg", "-an", "-readrate", "1", "-i", filename, "-f", "image2pipe", "pipe:1")
	}

	fmt.Println(strings.Join(cmd.Args, " "))

	// cmd := fluentffmpeg.NewCommand("").
	// 	InputPath(filename).
	// 	InputOptions("-r", "15", "-an", "-readrate", "1.1").
	// 	OutputFormat("image2pipe")

	writer := &AsciiResponseWriter{
		Context:      c,
		AsciiOptions: asciiOptions,
	}

	cmd.Stdout = writer

	// fmt.Println(cmd.Args)

	// cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		c.JSON(500, gin.H{
			"message": "Failed to process video",
		})
		return
	}

	ffmpegFinished := make(chan bool, 1)

	go func() {
		cmd.Wait()

		ffmpegFinished <- true
	}()

	cleanup := func(clientGone bool) {
		// prevent crash
		writer.Close(clientGone)

		if cmd.Process != nil {
			cmd.Process.Kill()
		}

		os.Remove(filename)
		if subtitleFile != "" {
			os.Remove(subtitleFile)
		}

	}

	for {
		select {
		case <-c.Request.Context().Done():
			// client https://i.imgflip.com/1h2kbp.jpg
			cleanup(true)
			return

		case <-ffmpegFinished:
			// fmt.Println("Done")
			c.Writer.Write([]byte("Thanks for watching!\n"))
			c.Writer.Flush()

			c.Status(200)

			cleanup(false)
			return
		}
		// https://media1.tenor.com/m/81vog8pvoaoAAAAC/ffxiv-frieren.gif
	}

}
