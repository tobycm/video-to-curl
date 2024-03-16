package main

import (
	"github.com/gin-gonic/gin"

	env "github.com/Netflix/go-env"
)

type Environment struct {
	ListenAddress string `env:"ADDRESS"`
}

var (
	environment Environment
)

func runServer() {
	router := gin.New()

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to tobycm's video to cURL service :)",
		})
	})

	if environment.ListenAddress == "" {
		environment.ListenAddress = ":80"
	}
	router.Run(environment.ListenAddress)
}

func main() {
	if _, err := env.UnmarshalFromEnviron(&environment); err != nil {
		panic(err)
	}

	runServer()
}
