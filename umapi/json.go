package umapi

import (
	"github.com/cosmouser/mudwork/config"
	"github.com/cosmouser/mudwork/ldapsearch"
	log "github.com/sirupsen/logrus"
)

type Items []Item
type Item struct {
	Do         []Action `json:"do"`
	Domain     string   `json:"domain,omitempty"`
	UseAdobeID bool     `json:"useAdobeID,omitempty"`
	User       string   `json:"user"`
}
type Action struct {
	AddAdobeID  *ActionAddAdobeID  `json:"addAdobeID,omitempty"`
	CreateFedID *ActionCreateFedID `json:"createFederatedID,omitempty"`
	Add         *ActionAdd         `json:"add,omitempty"`
	Remove      *ActionRemove      `json:"remove,omitempty"`
}
type ActionAdd struct {
	Group []string `json:"group"`
}
type ActionRemove struct {
	Group []string `json:"group"`
}
type ActionAddAdobeID struct {
	Country   string `json:"country,omitempty"`
	Email     string `json:"email"`
	FirstName string `json:"firstname,omitempty"`
	LastName  string `json:"lastname,omitempty"`
	Option    string `json:"option,omitempty"`
}
type ActionCreateFedID struct {
	Country   string `json:"country"`
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Option    string `json:"option,omitempty"`
}
type GroupResponse struct {
	LastPage bool    `json:"lastPage"`
	Result   string  `json:"result"`
	Groups   []Group `json:"groups"`
}
type Group struct {
	Type               string `json:"type"`
	GroupName          string `json:"groupName"`
	MemberCount        int    `json:"memberCount"`
	AdminGroupName     string `json:"adminGroupName,omitempty"`
	LicenseQuota       string `json:"licenseQuota,omitempty"`
	ProductName        string `json:"productName,omitempty"`
	ProductProfileName string `json:"productProfileName,omitempty"`
}
type ActionResponse struct {
	Completed           int                      `json:"completed"`
	NotCompleted        int                      `json:"notCompleted"`
	CompletedInTestMode int                      `json:"completedInTestMode"`
	Errors              *[]ActionResponseError   `json:"errors"`
	Result              string                   `json:"result"`
	Warnings            *[]ActionResponseWarning `json:"warnings"`
}
type ActionResponseError struct {
	Index     int    `json:"index"`
	Step      int    `json:"step"`
	RequestID string `json:"requestID"`
	Message   string `json:"message"`
	User      string `json:"user"`
	ErrorCode string `json:"errorCode"`
}
type ActionResponseWarning struct {
	WarningCode string `json:"warningCode"`
	RequestID   string `json:"requestID"`
	Index       int    `json:"index"`
	Step        int    `json:"step"`
	Message     string `json:"message"`
	User        string `json:"user"`
}

// The GenAddItem and GenRemoveItem functions are for creating structs
// to be appended to a slice of Items. The slice of Items is then
// marshalled into json and sent to the Adobe endpoint as a request body

// GenAddItem creates an Item for adding a user to a group. It also
// creates a federated ID for the user if one does not already exist
func GenAddItem(user, group string) Item {
	groupSlice := []string{group}
	addAction := &ActionAdd{groupSlice}
	person, err := ldapsearch.GetPerson(user)
	if err != nil {
		log.WithFields(log.Fields{
			"user": user,
		}).Warn("Ldap search failed")
	}
	createAction := &ActionCreateFedID{"US", person.Email, person.FirstName, person.LastName, "ignoreIfAlreadyExists"}
	actions := Action{Add: addAction, CreateFedID: createAction}
	uac := []Action{actions}
	item := Item{User: person.Email, Do: uac}
	return item
}

// GenRemoveItem creates an Item for removing a user from a group
func GenRemoveItem(user, group string) Item {
	groupSlice := []string{group}
	removeAction := &ActionRemove{groupSlice}
	action := Action{Remove: removeAction}
	uac := []Action{action}
	item := Item{User: user, Do: uac, Domain: config.C.Enterprise["Domain"]}
	return item
}

func GenAddRequest(user, group string) Items {
	uaaGroup := []string{group}
	uaa := &ActionAdd{uaaGroup}
	ua := Action{Add: uaa}
	uac := []Action{ua}
	ui := Item{User: user, Do: uac, Domain: config.C.Enterprise["Domain"]}
	umapiCollection := []Item{ui}
	return umapiCollection
}
