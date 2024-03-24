package utils

import (
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/qeesung/image2ascii/convert"
)

func ServeVideo(c *gin.Context, filename string, asciiOptions *convert.Options) {
	cmd := exec.Command("ffmpeg", "-an", "-readrate", "1", "-i", filename, "-f", "image2pipe", "pipe:1")

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
