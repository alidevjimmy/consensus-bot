package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// var token = os.Getenv("BotToken")
// var botID = os.Getenv("BotID")

var token = "1635980318:AAE0P8JTY5DSPiFEgOlqGKvO4EWq314dZlA"
var botID = "@simorgh_consensus_bot"

type webhookReqBody struct {
	Message struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

func Handler(res http.ResponseWriter, req *http.Request) {
	body := &webhookReqBody{}
	if err := json.NewDecoder(req.Body).Decode(body); err != nil {
		fmt.Println("could not decode request body", err)
		return
	}
	fmt.Println(body)
	if !strings.Contains(strings.ToLower(body.Message.Text), botID) {
		return
	}

	if err := ReactionHandler(body); err != nil {
		fmt.Println("Error:", err)
		return
	}
}

type sendMessageReqBody struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func ReactionHandler(rb *webhookReqBody) error {
	voteMatch, err := regexp.MatchString(botID+" delete -r .*", rb.Message.Text)
	if err != nil {
		return err
	}
	if voteMatch {
		return CreateVote(rb)
	}
	return nil
}

func CreateVote(rb *webhookReqBody) error {
	return SendMessage(rb.Message.Chat.ID, "vote created!")
}

func SendMessage(chatID int64, text string) error {
	reqBody := &sendMessageReqBody{
		ChatID: chatID,
		Text:   text,
	}

	sendBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected problem!")
	}
	return nil
}

func main() {
	fmt.Println("server is running...")
	http.ListenAndServe(":3000", http.HandlerFunc(Handler))
}
