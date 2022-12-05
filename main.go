package main

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type UsersResponse struct {
	Data []User `json:"data"`
}

type TweetsResponse struct {
	Data     []Tweet  `json:"data"`
	Metadata Metadata `json:"meta"`
}

type User struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type Tweet struct {
	CreatedAt string `json:"created_at"`
	Id        string `json:"id"`
	Text      string `json:"text"`
}

type Metadata struct {
	NewestId    string `json:"newest_id"`
	NextToken   string `json:"next_token"`
	OldestId    string `json:"oldest_id"`
	ResultCount int    `json:"result_count"`
}

func getTwitterUserId(client *http.Client, username, token string) (string, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/by?usernames=%s", username)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	r := &UsersResponse{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return "", err
	}
	return r.Data[0].Id, nil
}

func getTweets(client *http.Client, token, userId string) ([]Tweet, error) {
	tweets := make([]Tweet, 0, 1)
	url := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets?max_results=10&tweet.fields=created_at", userId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return tweets, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return tweets, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tweets, err
	}
	r := &TweetsResponse{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return tweets, err
	}
	for _, t := range r.Data {
		tweets = append(tweets, t)
	}
	return tweets, nil
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	bearerToken := os.Getenv("BEARER_TOKEN")
	twitterUsername := os.Getenv("TWITTER_USERNAME")

	if bearerToken == "" {
		log.Fatal().Msg("BEARER_TOKEN is required")
	}
	if twitterUsername == "" {
		log.Fatal().Msg("TWITTER_USERNAME is required")
	}

	client := &http.Client{}

	twitterUserId, err := getTwitterUserId(client, twitterUsername, bearerToken)
	if err != nil {
		log.Fatal().Err(err).Msg("Twitter user not found")
	}

	log.Info().Msgf("twitter username = %s", twitterUsername)
	log.Info().Msgf("twitter user id = %s", twitterUserId)

	last_id := ""

	for {
		tweets, err := getTweets(client, bearerToken, twitterUserId)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get tweet")
		}
		for _, t := range tweets {
			if strings.Compare(last_id, t.Id) < 0 {
				last_id = t.Id
				log.Info().Str("tweet-id", t.Id).Str("created-at", t.CreatedAt).Msg(t.Text)
			}
		}
		time.Sleep(300 * time.Second)
	}
}
