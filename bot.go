package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
)

///******************* Only Admins can use this robot *************************////
// only admins can vote
// hyperlink
// trace polls
// forward message to specific channel
// check message is vote or not
// creating function to audit polls (global variable)
// unpin message after completeing poll
// dockerizing

var token string
var botID string
var port string
var chanID string
var polls []PollType

type PollType struct {
	ID          int64
	AcceptCount int
	RejectCount int
	PinID       int64
}

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

type PollAnswer struct {
	PollID    string  `json:"poll_id"`
	User      User    `json:"user"`
	OptionIDs []int64 `json:"option_ids"`
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
	Ok     bool `json:"ok"`
	Result struct {
		User   User   `json:"user"`
		Status string `json:"status"`
	} `json:"result"`
}

type PollMessageRes struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageID int64 `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		User User `json:"from"`
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

type Update struct {
	UpdateID   int64      `json:"update_id"`
	Poll       Poll       `json:"poll"`
	PollAnswer PollAnswer `json:"poll_answer"`
}

type ForwardMessageReqBody struct {
	ChatID     int64 `json:"chat_id"`
	FromChatID int64 `json:"from_chat_id"`
	MessageID  int64 `json:"message_id"`
}

func ReactionHandler(rb *webhookReqBody) error {
	voteMatch, err := regexp.MatchString("@"+botID+" delete -r .*", rb.ReqMessage.Text)
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

	// adding note that only send vote on group
	// adding channel name is description
	// discution

	channelID, err := GetChannelID()
	if err != nil {
		return err
	}
	msg := fmt.Sprintf(`رأی به حذف %s به دلیل: %s`, `\\[inline URL\\]\\(http://www.example.com/\\)`, reason)
	pollRes, err := CreatePoll(rb.ReqMessage.Chat.ID, msg)
	if err != nil && pollRes == nil {
		return err
	}
	// fmt.Sprintf("%d",rb.ReqMessage.ReplyToMessage.MessageID)

	if err := PinChatMessage(pollRes.Result.Chat.ID, pollRes.Result.MessageID); err != nil {
		return err
	}

	if err := ForwardMessage(channelID, rb.ReqMessage.Chat.ID, rb.ReqMessage.ReplyToMessage.MessageID); err != nil {
		return err
	}

	// add poll_id in global var
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

func printJson(r *io.Reader) {
	body, err := ioutil.ReadAll(*r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(body)
}

func CreatePoll(chatID int64, question string) (*PollMessageRes, error) {
	sendPollReqBody := &SendPollReqBody{
		ChatID:      chatID,
		Question:    question,
		Options:     []string{"موافق", "مخالف"},
		IsAnonymous: false,
	}

	sendBody, err := json.Marshal(sendPollReqBody)
	if err != nil {
		return nil, err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendPoll", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println(res.Status)
		return nil, errors.New("unexpected error")
	}
	resBody := &PollMessageRes{}
	if err := json.NewDecoder(res.Body).Decode(resBody); err != nil {
		return nil, err
	}

	polls = append(polls, PollType{
		ID:          resBody.Result.MessageID,
		AcceptCount: 0,
		RejectCount: 0,
	})
	return resBody, nil
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
		return errors.New("1: unexpected error")
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

func GetAdmins(chatID int64) ([]User, error, int) {
	res, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getChatAdministrators?chat_id=%d", token, chatID))
	if err != nil {
		return nil, err, 0
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected error"), 0
	}

	defer res.Body.Close()

	type Data struct {
		Result []struct {
			User User `json:"user"`
		} `json:"result"`
	}
	resBody := &Data{}
	if err := json.NewDecoder(res.Body).Decode(resBody); err != nil {
		return nil, err, 0
	}
	users := []User{}
	for _, v := range resBody.Result {
		users = append(users, v.User)
	}
	return users, nil, len(users)
}

func GetUpdated() (*PollAnswer, error) {
	res, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getWebhookInfo", token))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected error")
	}

	defer res.Body.Close()
	pollAnswer := &PollAnswer{}
	err = json.NewDecoder(res.Body).Decode(pollAnswer)

	if err != nil {
		return nil, err
	}
	fmt.Println(pollAnswer)
	return pollAnswer, nil
}

func ForwardMessage(chatID, forwardChetID, messageID int64) error {
	forwardMessageReqBody := &ForwardMessageReqBody{
		ChatID:     chatID,
		FromChatID: forwardChetID,
		MessageID:  messageID,
	}

	sendBody, err := json.Marshal(forwardMessageReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/forwardMessage", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println(res.Status)
		body, _ := ioutil.ReadAll(res.Body)
		fmt.Println(string(body))
		return errors.New("1: unexpected error")
	}

	return nil
}

func GetChannelID() (int64, error) {
	res, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%s", token, chanID))
	if err != nil {
		return -1, err
	}
	if res.StatusCode != http.StatusOK {
		return -1, errors.New("unexpected error")
	}

	defer res.Body.Close()
	type Chat struct {
		Result struct {
			ID int64 `json:"id"`
		} `json:"result"`
	}

	chat := &Chat{}
	if err := json.NewDecoder(res.Body).Decode(chat); err != nil {
		return -1, err
	}

	return chat.Result.ID, nil
}

func Handler(res http.ResponseWriter, req *http.Request) {
	body := &webhookReqBody{}
	if err := json.NewDecoder(req.Body).Decode(body); err != nil {
		fmt.Println("1: could not decode request body", err)
		return
	}

	t, _ := json.Marshal(body)
	fmt.Println(string(t))
	// printJson(res)
	// fmt.Println(body)
	// GetUpdated()

	// check if message is vote run GetAdmins
	if !strings.Contains(body.ReqMessage.Text, "@"+botID) || reflect.ValueOf(body.ReqMessage.ReplyToMessage).IsZero() {
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

func getBot() error {
	res, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("Error while geting bot information")
	}

	defer res.Body.Close()

	type Data struct {
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	resBody := &Data{}
	if err := json.NewDecoder(res.Body).Decode(resBody); err != nil {
		return err
	}
	botID = resBody.Result.Username
	return nil
}

func setEnv() {
	if len(os.Args) < 4 || len(os.Args) > 4 {
		log.Fatal("Usage: COMMAND [token] [port] [channelID] note: channel_id should begin with @")
	}
	token = os.Args[1]
	port = os.Args[2]
	chanID = os.Args[3]
}

func main() {
	setEnv()
	getBot()
	fmt.Println("server is running...")

	http.ListenAndServe(":"+port, http.HandlerFunc(Handler))
}
