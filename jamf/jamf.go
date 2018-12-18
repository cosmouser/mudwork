package jamf

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/cosmouser/mudwork/config"
	"github.com/cosmouser/mudwork/data"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type JamfWebhook struct {
	Event   Event   `json:"event"`
	Webhook Webhook `json:"webhook"`
}
type Event struct {
	AuthorizedUsername   string `json:"authorizedUsername"`
	ObjectID             int    `json:"objectID"`
	ObjectName           string `json:"objectName"`
	ObjectTypeName       string `json:"objectTypeName"`
	OperationSuccessful  bool   `json:"operationSuccessful"`
	RestAPIOperationType string `json:"restAPIOperationType"`
}
type Webhook struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	WebhookEvent string `json:"webhookEvent"`
}

type AdvSearch struct {
	XMLName   xml.Name   `xml:"advanced_computer_search"`
	Computers []Computer `xml:"computers>computer"`
}

type Computer struct {
	Username string `xml:"Username"`
}

var (
	advSearchErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mudwork_advsearch_errors_total",
			Help: "Total number of errors when requesting search results from the JSS.",
		},
	)
)

// WebhookHandler is the http server handler for incoming Jamf Webhooks
func MakeWebhookHandler(messenger chan int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var jamfWebhook JamfWebhook
		// // Only respond to requests from the config.JssIP
		// if realIP := r.Header.Get("X-Real-IP"); realIP != config.JssIP {
		// 	w.WriteHeader(http.StatusForbidden)
		// 	w.Write([]byte(http.StatusText(http.StatusForbidden) + "\n"))
		// 	return
		// }
		switch r.Method {
		case "GET":
			w.Write([]byte("mudwork"))
		case "POST":
			defer r.Body.Close()
			err := json.NewDecoder(r.Body).Decode(&jamfWebhook)
			if err != nil {
				log.WithFields(log.Fields{
					"xrealip": r.Header.Get("X-Real-IP"),
				}).Warn(err)
				return
			}
			if jamfUser := jamfWebhook.Event.AuthorizedUsername; jamfUser != config.C.CirrupUser {
				//		log.WithFields(log.Fields{
				//			"webhook_id": jamfWebhook.Webhook.ID,
				//			"jamf_user":  jamfUser,
				//			"xrealip":    r.Header.Get("X-Real-IP"),
				//		}).Warn("user doesn't match cirrup user in config")
				return
			}
			// So far, we've confirmed with some certainty that the request
			// is from the JSS and is a POST in RestAPIOperation webhook
			// format and that it is from Cirrup.
			// Now, Mudwork should query its advanced search at the JSS for
			// a snapshot of the current list of users that should be given
			// entitlements.

			// Add an incremental backoff when errors received. Fail after a number of tries
			var gasnRetries int
			names, err := GetAdvSearchNames()
			if err != nil {
				log.WithFields(log.Fields{
					"function": "GetAdvSearchNames",
					"error":    err,
				}).Error("Unable to unmarshal response from JSS")
				advSearchErrors.Inc()
				for err != nil {
					gasnRetries++
					names, err = GetAdvSearchNames()
					if err != nil {
						log.WithFields(log.Fields{
							"function": "GetAdvSearchNames",
							"error":    err,
						}).Error("Unable to unmarshal response from JSS")
						advSearchErrors.Inc()
						// fail after a number of retries
						if gasnRetries > 5 {
							return
						}
						time.Sleep(time.Second * 4 * time.Duration(gasnRetries))
					}
				}
			}
			users := data.GetUsers()
			add := data.Diff(names, users)
			remove := data.Diff(users, names)
			var queuedAdd, queuedRemove, dupAdd, dupRemove int

			for _, j := range add {
				// filter out usernames less than 2 characters long
				if len(j) < 2 {
					continue
				}
				entry := &data.TxEntry{j, "add"}
				if inTxlog := data.LookupTxEntry(entry); !inTxlog {
					err := data.InsertTxEntry(entry)
					if err != nil {
						log.WithFields(log.Fields{
							"user":   entry.UniqueID,
							"method": "add",
							"table":  "txlog",
						}).Warn("Could not insert user")
					} else {
						queuedAdd++
					}
				} else {
					dupAdd++
				}
			}
			for _, j := range remove {
				// filter out usernames less than 2 characters long
				if len(j) < 2 {
					continue
				}
				entry := &data.TxEntry{j, "remove"}
				if inTxlog := data.LookupTxEntry(entry); !inTxlog {
					err := data.InsertTxEntry(entry)
					if err != nil {
						log.WithFields(log.Fields{
							"user":   entry.UniqueID,
							"method": "remove",
							"table":  "txlog",
						}).Warn("Could not insert user")
					} else {
						queuedRemove++
					}
				} else {
					dupRemove++
				}
			}
			numChanges := queuedAdd + queuedRemove + dupAdd + dupRemove
			log.WithFields(log.Fields{
				"total":         numChanges,
				"add_queued":    queuedAdd,
				"remove_queued": queuedRemove,
				"dup_add":       dupAdd,
				"dup_remove":    dupRemove,
			}).Info("Search parsed")
			if numChanges > 0 {
				messenger <- numChanges
			}
		}
	}
}

func GetNames(computers []Computer) []string {
	result := []string{}
	names := make(map[string]bool)
	for _, j := range computers {
		if _, ok := names[j.Username]; !ok {
			result = append(result, j.Username)
			names[j.Username] = true
		}
	}
	return result
}

func GetAdvSearchNames() ([]string, error) {
	result := AdvSearch{}
	resourceURI := fmt.Sprintf("%s/JSSResource/advancedcomputersearches/id/%d",
		config.C.JssUrl,
		config.C.AdvSearchID,
	)
	var client = &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", resourceURI, nil)
	req.SetBasicAuth(config.C.ApiUser, config.C.ApiPass)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	xmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal(xmlData, &result)
	if err != nil {
		return nil, err
	}
	names := GetNames(result.Computers)
	return names, nil
}
func init() {
	prometheus.MustRegister(advSearchErrors)
}
