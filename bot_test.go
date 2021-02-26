package main

import (
	"testing"
)

func TestGetAdmins(t *testing.T) {
	var chatID int64 = -1001339015148
	admins, err, count := GetAdmins(chatID)
	if err != nil {
		t.Error(err)
	}
	if count <= 0 {
		t.Error("admin counts is less or equal to 0")
	}
	if len(admins) <= 0 {
		t.Error("admins array is empty")
	}
}

func TestUnpinChatMessage(t *testing.T) {
	var chatID, messageID int64 = -1001339015148, 215
	if err := UnPinChatMessage(chatID, messageID); err != nil {
		t.Error("error: ", err)
	}

}
