package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// hyperlink
// dockerizing

var token string
var botID string
var port string
var chanID string
var polls = make(map[string]*PollType)

type PollType struct {
	AcceptCount        int
	RejectCount        int
	MessageUnderVoteID int64
	ChatID             int64
	PollMessageID      int64
	ReqMessageID       int64
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
	ChatID               int64    `json:"chat_id"`
	Question             string   `json:"question"`
	Options              []string `json:"options"`
	IsAnonymous          bool     `json:"is_anonymous"`
	ExplanationParseMode string   `json:"explanation_parse_mode"`
	ReplyToMessageID     int      `json:"reply_to_message_id"`
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
		Poll Poll `json:"poll"`
	} `json:"result"`
}

type webhookReqBody struct {
	ReqMessage Message `json:"message"`
}

type sendMessageReqBody struct {
	ChatID           int64  `json:"chat_id"`
	Text             string `json:"text"`
	ReplyToMessageID int64  `json:"reply_to_message_id"`
	ParseMode        string `json:"parse_mode"`
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
type DeletePollReqBody struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

// type Update struct {
// 	UpdateID   int64      `json:"update_id"`
// 	Poll       Poll       `json:"poll"`
// 	PollAnswer PollAnswer `json:"poll_answer"`
// }

type ForwardMessageReqBody struct {
	ChatID     int64 `json:"chat_id"`
	FromChatID int64 `json:"from_chat_id"`
	MessageID  int64 `json:"message_id"`
}

func ReactionHandler(update tgbotapi.Update) error {
	voteMatch, err := regexp.MatchString("@"+botID+" delete -r .*", update.Message.Text)
	if err != nil {
		return err
	}

	if voteMatch {
		return CreateVote(update)
	}
	return nil
}

func CreateVote(update tgbotapi.Update) error {
	// append(messagesUnderVote, rb.ReqMessage.ReplyToMessage.MessageID)
	pattern := regexp.MustCompile("-r (.*)")
	reason := pattern.FindStringSubmatch(update.Message.Text)[1]

	// adding note that only send vote on group
	// adding channel name is description
	// discution

	channelID, err := GetChannelID()
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("رأی به حذف پیام ریپلای شده " + "به دلیل: " + reason + fmt.Sprintf("\n\n\n\nبحث و گفتوگو‌: %s", chanID))
	pollRes, err := CreatePoll(update.Message.Chat.ID, int64(update.Message.ReplyToMessage.MessageID), int64(update.Message.MessageID), msg)
	if err != nil && pollRes == nil {
		return err
	}

	if err := PinChatMessage(pollRes.Result.Chat.ID, pollRes.Result.MessageID); err != nil {
		return err
	}

	if err := ForwardMessage(channelID, update.Message.Chat.ID, int64(update.Message.ReplyToMessage.MessageID)); err != nil {
		return err
	}

	return nil
}

func SendMessage(chatID int64, text string, replyToMessageID int64) error {
	reqBody := &sendMessageReqBody{
		ChatID:           chatID,
		Text:             text,
		ReplyToMessageID: replyToMessageID,
		ParseMode:        "MarkdownV2",
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
		return errors.New("error while delete message")
	}

	return nil
}

func DeletePoll(chatID, pollID int64) error {
	deletePollReqBody := &DeletePollReqBody{
		ChatID:    chatID,
		MessageID: pollID,
	}

	sendBody, err := json.Marshal(deletePollReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("error while delete poll")
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

func CreatePoll(chatID, messageID, reqMessageID int64, question string) (*PollMessageRes, error) {
	sendPollReqBody := &SendPollReqBody{
		ChatID:               chatID,
		Question:             question,
		Options:              []string{"موافق", "مخالف"},
		IsAnonymous:          false,
		ExplanationParseMode: "MarkdownV2",
		ReplyToMessageID:     int(messageID),
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
		return nil, errors.New("unexpected error while creating poll")
	}
	resBody := &PollMessageRes{}
	if err := json.NewDecoder(res.Body).Decode(resBody); err != nil {
		return nil, err
	}

	polls[resBody.Result.Poll.ID] = &PollType{
		AcceptCount:        0,
		RejectCount:        0,
		MessageUnderVoteID: messageID,
		ChatID:             chatID,
		PollMessageID:      resBody.Result.MessageID,
		ReqMessageID:       reqMessageID,
	}
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

func PinChatMessage(chatID, messageID int64) error {
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
		return errors.New("1: unexpected error while pin message")
	}
	return nil
}
func UnPinChatMessage(chatID, messageID int64) error {
	pinReqBody := &PinMessageReq{
		ChatID:    chatID,
		MessageID: messageID,
	}

	sendBody, err := json.Marshal(pinReqBody)
	if err != nil {
		return err
	}
	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/unpinChatMessage", token), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("error while unpin message")
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
		return nil, errors.New("unexpected error while geting admins data"), 0
	}

	defer res.Body.Close()

	type Data struct {
		Ok     bool `json:"ok"`
		Result []struct {
			User User `json:"user"`
		} `json:"result"`
	}
	resBody := &Data{}
	if err := json.NewDecoder(res.Body).Decode(resBody); err != nil {
		return nil, err, 0
	}
	if !resBody.Ok {
		return nil, errors.New("cannot get admins"), 0
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
		return nil, errors.New("unexpected error while getting updates")
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
		return errors.New("1: unexpected error while forward message")
	}

	return nil
}

func GetChannelID() (int64, error) {
	res, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%s", token, chanID))
	if err != nil {
		return -1, err
	}
	if res.StatusCode != http.StatusOK {
		return -1, errors.New("unexpected error while getting channel id")
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

// Handler handle incoming messages
func Handler(update tgbotapi.Update) {
	if !reflect.ValueOf(update.PollAnswer).IsZero() {
		if _, exists := polls[update.PollAnswer.PollID]; !exists {
			return
		}
		admins, err, count := GetAdmins(polls[update.PollAnswer.PollID].ChatID)
		if err != nil || len(admins) <= 0 || count <= 0 {
			fmt.Println("Error: ", "error in getting admins proccess", err, len(admins), count)
			return
		}
		isAdmin := false
		for _, v := range admins {
			if v.ID == int64(update.PollAnswer.User.ID) {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			return
		}
		// increase poll vote counts
		if update.PollAnswer.OptionIds[0] == 0 {
			polls[update.PollAnswer.PollID].AcceptCount++
		} else if update.PollAnswer.OptionIds[0] == 1 {
			polls[update.PollAnswer.PollID].RejectCount++

		}
		// check poll is completed or not
		isAcceptedOrRejected := false
		if polls[update.PollAnswer.PollID].AcceptCount >= int(math.Ceil(float64(count)/3)) {
			isAcceptedOrRejected = true
			DeleteMessage(polls[update.PollAnswer.PollID].ChatID, polls[update.PollAnswer.PollID].MessageUnderVoteID)
		} else if polls[update.PollAnswer.PollID].RejectCount >= int(math.Ceil(float64(count)/3)) {
			isAcceptedOrRejected = true
		}
		if isAcceptedOrRejected {
			UnPinChatMessage(polls[update.PollAnswer.PollID].ChatID, polls[update.PollAnswer.PollID].PollMessageID)
			DeletePoll(polls[update.PollAnswer.PollID].ChatID, polls[update.PollAnswer.PollID].PollMessageID)
			DeleteMessage(polls[update.PollAnswer.PollID].ChatID, polls[update.PollAnswer.PollID].ReqMessageID)
			DeleteMessage(polls[update.PollAnswer.PollID].ChatID, polls[update.PollAnswer.PollID].PollMessageID + 1)
		}
		return
	}

	if !strings.Contains(update.Message.Text, "@"+botID) || reflect.ValueOf(update.Message.ReplyToMessage).IsZero() {
		return
	}
	if update.Message.From.IsBot {
		return
	}

	// thanks to vsCode :)
	chatMember, err := GetChatMember(update.Message.Chat.ID, int64(update.Message.From.ID))

	if err != nil || chatMember == nil {
		return
	}
	if chatMember.Result.Status != "creator" && chatMember.Result.Status != "administrator" {
		return
	}
	if err := ReactionHandler(update); err != nil {
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
	if len(os.Args) < 3 || len(os.Args) > 3 {
		log.Fatal("Usage: COMMAND [token] [channelID] note: channel_id should begin with @")
	}
	token = os.Args[1]
	chanID = os.Args[2]
}

func main() {
	setEnv()
	getBot()

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil && update.PollAnswer == nil {
			continue
		}

		Handler(update)
	}
}
