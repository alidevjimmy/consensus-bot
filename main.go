package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

///******************* Only Admins can use this robot *************************////

// var token = os.Getenv("BotToken")
// var botID = os.Getenv("BotID")

var token = "1635980318:AAE0P8JTY5DSPiFEgOlqGKvO4EWq314dZlA"
var botID = "@simorgh_consensus_bot"

// var messagesUnderVote = []int64{}
// should be map
var openPolls = []int64{}

type Message struct {
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	From      User   `json:"from"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`

	ReplyToMessage struct {
		MessageID int64  `json:"message_id"`
		Text      string `json:"text"`
		From      User   `json:"user"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"reply_to_message"`
}

type Poll struct {
	ID              string       `json:"id"`
	Question        string       `json:"question"`
	Options         []PollOption `json:"options"`
	TotalVoterCount uint         `json:"total_voter_count"`
	IsClosed        bool         `json:"is_closed"`
}

type PollOption struct {
	Text       string `json:"text"`
	VoterCount uint   `json:"voter_count"`
}

type SendPollReqBody struct {
	ChatID      int64    `json:"chat_id"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	IsAnonymous bool     `json:"is_anonymous"`
}

type StopPollReqBody struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

type User struct {
	ID    int64 `json:"id"`
	IsBot bool  `json:"is_bot"`
}

type ChatMember struct {
	Ok bool `json:"ok"`
	Result struct {
		User   User   `json:"user"`
		Status string `json:"status"`
	} `json:"result"`
}

type webhookReqBody struct {
	ReqMessage Message `json:"message"`
}

type sendMessageReqBody struct {
	ChatID           int64  `json:"chat_id"`
	Text             string `json:"text"`
	ReplyToMessageID int64  `json:"reply_to_message_id"`
}

type PinMessageReq struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

type GetChatMemberReq struct {
	ChatID int64 `json:"chat_id"`
	UserID int64 `json:"user_id"`
}

type DeleteMessageReqBody struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

func ReactionHandler(rb *webhookReqBody) error {
	voteMatch, err := regexp.MatchString(botID+" delete -r .*", rb.ReqMessage.Text)
	if err != nil {
		return err
	}

	if voteMatch {
		return CreateVote(rb)
	}
	return nil
}

func CreateVote(rb *webhookReqBody) error {
	// append(messagesUnderVote, rb.ReqMessage.ReplyToMessage.MessageID)
	pattern := regexp.MustCompile("-r (.*)")
	reason := pattern.FindStringSubmatch(rb.ReqMessage.Text)[1]
	err := CreatePoll(rb.ReqMessage.Chat.ID, fmt.Sprintf("رأی به پاک کردن %s به دلیل: %s", "[inline URL](http://www.example.com/)", reason))
	// fmt.Sprintf("%d",rb.ReqMessage.ReplyToMessage.MessageID)
	if err != nil {
		return err
	}
	// Pin Poll
	// err := PinChatMessage();
	return nil
}

func SendMessage(chatID int64, text string, replyToMessageID int64) error {
	reqBody := &sendMessageReqBody{
		ChatID:           chatID,
		Text:             text,
		ReplyToMessageID: replyToMessageID,
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
		return errors.New("unexpected problem")
	}
	return nil
}

func DeleteMessage(chatID int64, messageID int64) error {
	deleteMessageReqBody := &DeleteMessageReqBody{
		ChatID:    chatID,
		MessageID: messageID,
	}

	sendBody, err := json.Marshal(deleteMessageReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected error")
	}

	return nil
}

func CreatePoll(chatID int64, question string) error {
	sendPollReqBody := &SendPollReqBody{
		ChatID:      chatID,
		Question:    question,
		Options:     []string{"Accept", "Reject"},
		IsAnonymous: false,
	}

	sendBody, err := json.Marshal(sendPollReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendPoll", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected error")
	}

	return nil
}

func StopPoll(chatID int64, messageID int64) error {
	stopPollReqBody := &StopPollReqBody{
		ChatID:    chatID,
		MessageID: messageID,
	}

	sendBody, err := json.Marshal(stopPollReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/stopPoll", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected error")
	}

	return nil
}

func PinChatMessage(chatID int64, messageID int64) error {
	pinReqBody := &PinMessageReq{
		ChatID:    chatID,
		MessageID: messageID,
	}

	sendBody, err := json.Marshal(pinReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/pinChatMessage", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected error")
	}

	return nil
}

func GetChatMember(chatID int64, userID int64) (*ChatMember, error) {
	res, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember?chat_id=%d&user_id=%d", token, chatID, userID))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected error")
	}

	defer res.Body.Close()

	chatMember := &ChatMember{}
	err = json.NewDecoder(res.Body).Decode(chatMember)

	if err != nil {
		return nil, err
	}
	return chatMember, nil
}

func Handler(res http.ResponseWriter, req *http.Request) {
	body := &webhookReqBody{}
	if err := json.NewDecoder(req.Body).Decode(body); err != nil {
		fmt.Println("could not decode request body", err)
		return
	}
	fmt.Println(body)
	if !strings.Contains(strings.ToLower(body.ReqMessage.Text), botID) || reflect.ValueOf(body.ReqMessage.ReplyToMessage).IsZero() {
		return
	}

	if body.ReqMessage.From.IsBot {
		return
	}
	chatMember, err := GetChatMember(body.ReqMessage.Chat.ID, body.ReqMessage.From.ID)

	if err != nil || chatMember == nil {
		return
	}

	if chatMember.Result.Status != "creator" && chatMember.Result.Status != "administrator" {
		return
	}

	if err := ReactionHandler(body); err != nil {
		fmt.Println("Error:", err)
		return
	}
}

func main() {
	fmt.Println("server is running...")
	http.ListenAndServe(":3000", http.HandlerFunc(Handler))
}
