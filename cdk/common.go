package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func getMemorydbPassword() string {
	memorydbPassword := os.Getenv("MEMORYDB_PASSWORD")
	if memorydbPassword == "" {
		log.Fatalf("env var %s missing\n", "MEMORYDB_PASSWORD")
	}

	return memorydbPassword
}

func getTwitterAPIKey() string {
	twitterAPIKey := os.Getenv("TWITTER_API_KEY")
	if twitterAPIKey == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_API_KEY")
	}

	return twitterAPIKey
}

func getTwitterAPISecret() string {
	twitterAPISecret := os.Getenv("TWITTER_API_SECRET")
	if twitterAPISecret == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_API_SECRET")
	}
	return twitterAPISecret
}

func getTwitterAccessToken() string {
	twitterAccessToken := os.Getenv("TWITTER_ACCESS_TOKEN")
	if twitterAccessToken == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_ACCESS_TOKEN")
	}

	return twitterAccessToken
}

func getTwitterAccessTokenSecret() string {
	twitterAccessTokenSecret := os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
	if twitterAccessTokenSecret == "" {
		log.Fatalf("env var %s missing\n", "TWITTER_ACCESS_TOKEN_SECRET")
	}
	return twitterAccessTokenSecret
}

func _random(prefix string) string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := rng.Intn(1000) + 1
	snum := strconv.Itoa(num)
	return fmt.Sprintf("%s-%s", prefix, snum)
}
