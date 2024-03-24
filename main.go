package main

import (
	"github.com/gin-gonic/gin"
	"vsus.app/tobycm/video-to-curl/routes"

	env "github.com/Netflix/go-env"
)

type Environment struct {
	TempDir string `env:"TEMP_DIR"`

	ListenAddress string `env:"ADDRESS"`
}

var (
	environment Environment
)

func runServer() {
	router := gin.New()
	root := router.Group("/")

	root.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to tobycm's video to cURL service :)",
		})
	})

	watchRoute := root.Group("/watch")

	routes.AddWatchRoute(watchRoute, routes.WatchRouteOptions{
		TempDir: environment.TempDir,
	})

	if environment.ListenAddress == "" {
		environment.ListenAddress = ":3000"
	}

	router.Run(environment.ListenAddress)
}

func main() {
	if _, err := env.UnmarshalFromEnviron(&environment); err != nil {
		panic(err)
	}

	runServer()
}
