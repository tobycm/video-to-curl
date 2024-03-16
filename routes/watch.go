package routes

import (
	"bytes"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qeesung/image2ascii/convert"
)

type WatchRouteOptions struct {
	TempDir string
}

type PipeAndAsciifyToQueueWriter struct {
	queue        []string
	asciiOptions *convert.Options

	onShift func(q *PipeAndAsciifyToQueueWriter)
	onAdd   func(q *PipeAndAsciifyToQueueWriter)
}

func (writer *PipeAndAsciifyToQueueWriter) Write(p []byte) (n int, err error) {
	converter := convert.NewImageConverter()
	image, err := jpeg.Decode(bytes.NewReader(p))
	if err != nil {
		// fmt.Println(err.Error())
		// ignored
		return len(p), nil
	}
	ascii := converter.Image2ASCIIString(image, writer.asciiOptions)

	writer.queue = append(writer.queue, ascii)

	if writer.onAdd != nil {
		writer.onAdd(writer)
	}

	return len(p), nil
}

func (writer *PipeAndAsciifyToQueueWriter) Shift() string {
	if writer.onShift != nil {
		writer.onShift(writer)
	}

	frame := writer.queue[0]
	writer.queue = writer.queue[1:]

	return frame
}

func AddWatchRoute(router *gin.RouterGroup, options WatchRouteOptions) {
	router.POST("/watch/:name", func(c *gin.Context) {
		c.Writer.Write([]byte("Initiating...\n"))
		c.Writer.Flush()

		name := c.Param("name")

		width := c.Query("width")
		height := c.Query("height")

		if width == "" {
			width = "150"
		}

		if height == "" {
			height = "150"
		}

		if _, err := os.Stat(options.TempDir); os.IsNotExist(err) {
			if err := os.Mkdir(options.TempDir, 0777); err != nil {
				c.JSON(500, gin.H{
					"message": "Failed to create temp dir",
				})
				// fmt.Println(err.Error())
				return
			}
		}

		filename := options.TempDir + "/" + name + ".mp4"

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

		cmd := exec.Command("ffmpeg", "-r", "18", "-an", "-readrate", "1", "-i", filename, "-f", "image2pipe", "pipe:1")

		// cmd := fluentffmpeg.NewCommand("").
		// 	InputPath(filename).
		// 	InputOptions("-r", "15", "-an", "-readrate", "1.1").
		// 	OutputFormat("image2pipe")

		asciiOptions := convert.DefaultOptions
		if asciiOptions.FixedWidth, err = strconv.Atoi(width); err != nil {
			c.JSON(500, gin.H{
				"message": "Failed to parse width value",
			})
			return
		}

		if asciiOptions.FixedHeight, err = strconv.Atoi(height); err != nil {
			c.JSON(500, gin.H{
				"message": "Failed to parse height value",
			})
			return
		}

		// 150 raw frames buffer
		writer := &PipeAndAsciifyToQueueWriter{
			queue:        make([]string, 0, 300),
			asciiOptions: &asciiOptions,

			onShift: func(q *PipeAndAsciifyToQueueWriter) {
				if len(q.queue) >= 60 {
					return
				}

				if cmd.Process == nil {
					// fmt.Println("Process is nil but array is almost empty! this is no good...")
					return
				}

				// fmt.Println("Continuing process...")

				if err := cmd.Process.Signal(syscall.SIGCONT); err != nil {
					c.JSON(500, gin.H{
						"message": "Failed to continue process",
					})
					// fmt.Println(err.Error())
					return
				}
			},
			onAdd: func(q *PipeAndAsciifyToQueueWriter) {
				if len(q.queue) < 240 {
					return
				}

				if cmd.Process == nil {
					// fmt.Println("Process is nil but array is almost full! this is no good...")
					return
				}

				// fmt.Println("Pausing process...")

				if err := cmd.Process.Signal(syscall.SIGSTOP); err != nil {
					c.JSON(500, gin.H{
						"message": "Failed to stop process",
					})
					// fmt.Println(err.Error())
					return
				}
			},
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

		cleanup := func(clientGone bool) {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}

			if file != nil {
				file.Close()
				os.Remove(filename)
			}

			if clientGone {
				c.Abort()
			}

			cmd.Wait()
		}

		done := make(chan bool, 1)

		for {
			select {
			case <-c.Request.Context().Done():
				// client https://i.imgflip.com/1h2kbp.jpg
				cleanup(true)
				return

			case <-done:
				// fmt.Println("Done")
				c.Writer.Write([]byte("Thanks for watching!\n"))
				c.Writer.Flush()

				c.Status(200)

				cleanup(false)
				return
			default:
				if len(writer.queue) == 0 {
					if cmd.ProcessState == nil {
						continue
					}

					if cmd.ProcessState.Exited() {
						// fmt.Println("Process exited")
						done <- true
						return
					}

					continue
				}
				// fmt.Println(len(writer.queue))

				c.Writer.Write([]byte("\x1b[2J" + writer.Shift()))
				c.Writer.Flush()

				time.Sleep(time.Second / 20)

			}

		}

	})

}
