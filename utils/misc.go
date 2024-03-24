package utils

import (
	"bytes"
	"image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/qeesung/image2ascii/convert"
)

type AsciiResponseWriter struct {
	Context      *gin.Context
	AsciiOptions *convert.Options

	stop       bool
	clientGone bool
}

func (writer *AsciiResponseWriter) Write(p []byte) (n int, err error) {
	converter := convert.NewImageConverter()
	image, err := jpeg.Decode(bytes.NewReader(p))
	if err != nil {
		// fmt.Println(err.Error())
		// ignored
		return len(p), nil
	}
	ascii := converter.Image2ASCIIString(image, writer.AsciiOptions)

	if writer.Context.Writer == nil {
		return len(p), nil
	}

	if writer.stop {
		if writer.clientGone {
			writer.Context.Abort()
		}

		return len(p), nil
	}

	writer.Context.Writer.Write([]byte("\x1b[2J" + ascii))

	return len(p), nil
}

func (writer *AsciiResponseWriter) Close(clientGone bool) error {
	writer.stop = true

	writer.clientGone = clientGone

	return nil
}
