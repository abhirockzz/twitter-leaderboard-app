package main

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/go-redis/redis/v8"
)

const tweetCheckpointKey = "last_tweet_id"
const hashtagsSortedSet = "hashtags"
const twitterQuery = "aws"
const fetchTweetCount = 100

var memoryDBEndpoint string
var memoryDBUser string
var memoryDBPassword string

var twitterAPIKey string
var twitterAPISecret string
var twitterAccessToken string
var twitterAccessTokenSecret string

var rc *redis.ClusterClient

func init() {

	memoryDBEndpoint = os.Getenv("MEMORYDB_ENDPOINT")
	if memoryDBEndpoint == "" {
		log.Fatalf("env var %s missing\n", "MEMORYDB_ENDPOINT")
	}

	memoryDBUser = os.Getenv("MEMORYDB_USER")
	if memoryDBUser == "" {
		log.Fatalf("env var %s missing\n", "MEMORYDB_USER")
	}

	memoryDBPassword = os.Getenv("MEMORYDB_PASSWORD")
	if memoryDBPassword == "" {
		log.Fatalf("env var %s missing\n", "MEMORYDB_PASSWORD")
	}

	twitterAPIKey = os.Getenv("TWITTER_API_KEY")
	if twitterAPIKey == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_API_KEY")
	}

	twitterAPISecret = os.Getenv("TWITTER_API_SECRET")
	if twitterAPISecret == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_API_SECRET")
	}

	twitterAccessToken = os.Getenv("TWITTER_ACCESS_TOKEN")
	if twitterAccessToken == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_ACCESS_TOKEN")
	}

	twitterAccessTokenSecret = os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
	if twitterAccessTokenSecret == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_ACCESS_TOKEN_SECRET")
	}

	rc = redis.NewClusterClient(&redis.ClusterOptions{Addrs: []string{memoryDBEndpoint}, Username: memoryDBUser, Password: memoryDBPassword, TLSConfig: &tls.Config{}})

	err := rc.Ping(context.Background()).Err()
	if err != nil {
		log.Fatal("failed to connect", memoryDBEndpoint, err)
	}

	log.Println("connected to redis", memoryDBEndpoint)

}

func main() {
	lambda.Start(app)
}

func app() error {

	config := oauth1.NewConfig(twitterAPIKey, twitterAPISecret)
	token := oauth1.NewToken(twitterAccessToken, twitterAccessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	entities := true

	searchParams := &twitter.SearchTweetParams{
		Query: twitterQuery, IncludeEntities: &entities, Count: fetchTweetCount,
	}

	log.Println("fetching tweet checkpoint..")
	tweetIdCheckpoint, err := rc.Get(context.Background(), tweetCheckpointKey).Result()

	if err == nil {

		since, err := strconv.Atoi(tweetIdCheckpoint)
		if err != nil {
			log.Println("got invalid tweet id", err)
			return err
		}
		searchParams.SinceID = int64(since)
		log.Println("searching for tweets since", since)
	}

	search, _, err := client.Search.Tweets(searchParams)

	if err != nil {
		log.Println("tweets search failed", err)
		return err
	}

	tweets := search.Statuses

	log.Printf("got %v tweets\n", len(tweets))

	ctx := context.Background()
	pipeline := rc.Pipeline()

	for _, tweet := range tweets {
		if tweet.PossiblySensitive {
			log.Println("skipping possibly sensitive tweet", tweet.Text)
			continue
		}

		hashtagEntities := tweet.Entities.Hashtags
		for _, he := range hashtagEntities {
			hashtag := he.Text

			//just pipeline the zincrby
			err = pipeline.ZIncrBy(ctx, hashtagsSortedSet, 1, hashtag).Err()
			if err != nil {
				log.Println("redis pipeline zincr failed", err)
				return err
			}
			//err = rc.ZIncrBy(ctx, hashtagsSortedSet, 1, hashtag).Err()
			//log.Println("incremented/added", hashtag)
		}
	}

	//execute the pipeline
	cmds, err := pipeline.Exec(ctx)
	if err != nil {
		log.Println("redis pipeline exec failed", err)
		return err
	}

	for _, cmd := range cmds {
		if cmd.Err() == nil {
			log.Println("executed", cmd.Args())
		} else {
			log.Println("failed to execute", cmd.Args())
		}
	}

	// update checkpoint
	newCheckpointTweetID := tweets[len(tweets)-1].ID
	rc.Set(ctx, tweetCheckpointKey, strconv.Itoa(int(newCheckpointTweetID)), 0).Err()
	if err != nil {
		log.Println("failed to set checkpoint", err)
		return err
	}

	log.Println("set new checkpoint tweet to", newCheckpointTweetID)

	return nil
}
