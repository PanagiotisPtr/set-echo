package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	redis "github.com/go-redis/redis/v9"
)

func ProvideRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

type Payload struct {
	Value int64 `json:"value"`
}

var localState int64

const (
	stateKey = "state"
)

func ProvideRouter(
	rc *redis.Client,
) *gin.Engine {
	router := gin.Default()

	router.POST("/set", func(c *gin.Context) {
		ctx := c.Request.Context()
		var p Payload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		_, err := rc.Set(
			ctx,
			stateKey,
			p.Value,
			redis.KeepTTL,
		).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	router.POST("/sync", func(c *gin.Context) {
		ctx := c.Request.Context()
		val, err := rc.Get(
			ctx,
			stateKey,
		).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		localState, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	router.GET("/get", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"state": localState,
		})
	})

	return router
}

func main() {
	rc := ProvideRedisClient()
	router := ProvideRouter(rc)

	router.Run(":8888")
}
