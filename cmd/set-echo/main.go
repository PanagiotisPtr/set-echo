package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"log"

	"github.com/gin-gonic/gin"
	redis "github.com/go-redis/redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func ProvideKuberentesClientset() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	return kubernetes.NewForConfig(config)
}

func ProvideRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
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
	port     = 8888
)

func GetServiceEndpoints(
	ctx context.Context,
	client *kubernetes.Clientset,
	serviceName string,
) ([]string, error) {
	endpoints := []string{}
	serviceData, err := client.CoreV1().Endpoints(os.Getenv("POD_NAMESPACE")).Get(
		ctx,
		serviceName,
		metav1.GetOptions{},
	)
	if err != nil {
		return endpoints, err
	}

	for _, subset := range serviceData.Subsets {
		for _, address := range subset.Addresses {
			endpoints = append(endpoints, address.IP)
		}
	}

	return endpoints, nil
}

func ProvideRouter(
	rc *redis.Client,
	kc *kubernetes.Clientset,
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

		endpoints, err := GetServiceEndpoints(ctx, kc, os.Getenv("SERVICE_NAME"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		for _, endpoint := range endpoints {
			url := fmt.Sprintf("%s:%d/sync", endpoint, port)
			req, err := http.NewRequest(
				http.MethodPost,
				url,
				bytes.NewReader([]byte{}),
			)
			if err != nil {
				log.Printf("failed to make request for url %s", url)
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("failed to send request to url %s", url)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				log.Printf("did not receive 200 code from url %s", url)
			} else {
				log.Printf("successfully synced endpoint with IP %s", endpoint)
			}
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
	kc, err := ProvideKuberentesClientset()
	if err != nil {
		panic(err)
	}
	router := ProvideRouter(rc, kc)

	router.Run(fmt.Sprintf(":%d", port))
}
