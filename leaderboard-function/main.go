package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-redis/redis/v8"
)

var (
	clusterEndpoint string
	username        string
	password        string

	rc *redis.ClusterClient
)

func init() {
	clusterEndpoint = os.Getenv("MEMORYDB_ENDPOINT")
	if clusterEndpoint == "" {
		log.Fatal("MEMORYDB_ENDPOINT env var missing")
	}

	username = os.Getenv("MEMORYDB_USERNAME")
	if username == "" {
		log.Fatal("MEMORYDB_USERNAME env var missing")
	}

	password = os.Getenv("MEMORYDB_PASSWORD")
	if password == "" {
		log.Fatal("MEMORYDB_PASSWORD env var missing")
	}

	log.Println("connecting to memorydb cluster", clusterEndpoint)

	rc = redis.NewClusterClient(&redis.ClusterOptions{Username: username, Password: password,
		Addrs:     []string{clusterEndpoint},
		TLSConfig: &tls.Config{},
	})

	err := rc.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("failed to connect to memorydb redis. error message - %v", err)
	}

	log.Println("successfully connected to memorydb cluster", clusterEndpoint)
}

const hashtagsSortedSet = "hashtags"

func leaderboard(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	log.Println("fetching leaderboard info....")

	//top 10 hashtags
	hashtags, err := rc.ZRevRangeWithScores(context.Background(), hashtagsSortedSet, 0, 9).Result()

	//hashtags, err := rc.ZRangeArgsWithScores(context.Background(), redis.ZRangeArgs{Key: hashtagsSortedSet, Start: 0, Stop: -1, Rev: true, Count: 10}).Result()

	if err != nil {
		log.Println("failed to get info from sorted set using 'zrevrangewithscores'")
		return events.LambdaFunctionURLResponse{}, err
	}

	var lb []redis.Z

	for _, hashtag := range hashtags {
		lb = append(lb, hashtag)
	}

	hashtagsB, err := json.Marshal(lb)
	if err != nil {
		log.Println("failed to marshal hashtag info")
		return events.LambdaFunctionURLResponse{}, err
	}

	log.Println("successfully fetched leaderboard info....")

	return events.LambdaFunctionURLResponse{Body: string(hashtagsB), StatusCode: http.StatusOK}, nil

}

func main() {
	lambda.Start(leaderboard)
}
