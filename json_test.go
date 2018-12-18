package umapi

import (
	"encoding/json"
	"testing"
)

func TestActionResponse(t *testing.T) {
	actionResponseError := `
{
  "completed": 0,
  "notCompleted": 1,
  "completedInTestMode": 0,
  "errors": [
    {
      "index": 0,
      "step": 0,
      "message": "String too long in command for field: country, max length 2",
      "errorCode": "error.command.string.too_long"
    }
  ],
  "result": "error"
}`
	actionResponsePartial := `
	{
  "completed": 5,
  "notCompleted": 5,
  "completedInTestMode": 0,
  "errors": [
    {
      "index": 1,
      "step": 0,
      "requestID": "Two2_123456",
      "message": "User Id does not exist: test@test_fake.us",
      "user": "test@test_fake.us",
      "errorCode": "error.user.nonexistent"
    },
    {
      "index": 3,
      "step": 0,
      "requestID": "Four4_123456",
      "message": "Group NON_EXISTING_GROUP was not found",
      "user": "user4@example.com",
      "errorCode": "error.group.not_found"
    },
    {
      "index": 5,
      "step": 0,
      "requestID": "Six6_123456",
      "message": "User Id does not exist: test@test_fake.fake",
      "user": "test6@test_fake.fake",
      "errorCode": "error.user.nonexistent"
    },
    {
      "index": 7,
      "step": 0,
      "requestID": "Eight8_123456",
      "message": "Changes to users are only allowed in claimed domains.",
      "user": "fake8@faketest.com",
      "errorCode": "error.domain.trust.nonexistent"
    },
    {
      "index": 9,
      "step": 0,
      "requestID": "Ten10_123456",
      "message": "Group NON_EXISTING_GROUP was not found",
      "user": "user10@example.com",
      "errorCode": "error.group.not_found"
    }
  ],
  "result": "partial",
  "warnings": [
    {
      "warningCode": "warning.command.deprecated",
      "requestID": "Four4_123456",
      "index": 3,
      "step": 0,
      "message": "'product' command is deprecated. Please use productConfiguration.",
      "user": "user4@example.com"
    },
    {
      "warningCode": "warning.command.deprecated",
      "requestID": "Ten10_123456",
      "index": 9,
      "step": 0,
      "message": "'product' command is deprecated. Please use productConfiguration.",
      "user": "user10@example.com"
    }
  ]
}`
	actionResponseSuccess := `
	{
  "completed": 1,
  "notCompleted": 0,
  "completedInTestMode": 0,
  "result": "success"
}`

	arError := &ActionResponse{}
	arPartial := &ActionResponse{}
	arSuccess := &ActionResponse{}

	err := json.Unmarshal([]byte(actionResponseError), arError)
	if err != nil {
		t.Error(err)
	}
	want := "error"
	if got := arError.Result; got != want {
		t.Errorf("arError wanted %s, got %s\n", got, want)
	}
	err = json.Unmarshal([]byte(actionResponsePartial), arPartial)
	if err != nil {
		t.Error(err)
	}
	wantIndexes := []int{1, 3, 5, 7, 9}
	gotIndexes := []int{}
	for _, j := range *arPartial.Errors {
		gotIndexes = append(gotIndexes, j.Index)
	}
	for i, j := range gotIndexes {
		if j != wantIndexes[i] {
			t.Errorf("got %d, wanted %d\n", j, wantIndexes[i])
		}
	}
	err = json.Unmarshal([]byte(actionResponseSuccess), arSuccess)
	if err != nil {
		t.Error(err)
	}
	want = "success"
	if got := arSuccess.Result; got != want {
		t.Errorf("arSuccess wanted %s, got %s\n", got, want)
	}

}
