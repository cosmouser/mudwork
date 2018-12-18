package main

import (
	"encoding/json"
	"fmt"
	"github.com/cosmouser/mudwork/config"
	"github.com/cosmouser/mudwork/data"
	"github.com/cosmouser/mudwork/jamf"
	"github.com/cosmouser/mudwork/ldapsearch"
	"github.com/cosmouser/mudwork/umapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	responsesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mudwork_http_responses_total",
			Help: "Total number of responses from Adobe endpoints",
		},
		[]string{"status"},
	)
	dbSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mudwork_db_size_bytes",
		Help: "Current size of the Mudwork db in bytes",
	})
	managedAccounts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mudwork_db_users_rows",
		Help: "Number of users with entitlements managed through Mudwork",
	})
)

func init() {
	// Register the counters and gauges with Prometheus's default registry.
	prometheus.MustRegister(responsesTotal)
	prometheus.MustRegister(dbSize)
	prometheus.MustRegister(managedAccounts)
}

func main() {
	fmt.Fprint(ioutil.Discard, "Copyright (c) 2018, Regents of the University of California. All rights reserved.")
	if *config.FlagNoInit {
		log.Info("flag -noinit set, skipping token initialization")
	} else {
		umapi.Token.Renew()
	}
	if *config.FlagGroups {
		PrintGroups(umapi.Token)
		return
	}
	if *config.FlagTestMode {
		log.Info("testOnly set to true")
	}
	// prometheus db size gauge
	go func() {
		for {
			dbSize.Set(data.GetDBSize())
			managedAccounts.Set(float64(len(data.GetUsers())))
			time.Sleep(time.Second * 60)
		}
	}()
	msgs := make(chan int)
	go worker(msgs)
	handleWebhook := jamf.MakeWebhookHandler(msgs)
	http.HandleFunc("/mudwork", handleWebhook)
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", *config.FlagPort), nil)
}

func PrintGroups(token *umapi.AccessResponse) {
	groups, err := umapi.GetGroups(token)
	if err != nil {
		panic(err)
	}
	for _, j := range groups {
		log.Printf("%+v", j)
	}
}
func worker(messenger chan int) {
	for i := range messenger {
		log.WithFields(log.Fields{
			"num_changes": i,
		}).Info("changes received")
		processQueue()
	}
}
func processQueue() {
	txEntries, err := data.GetTxEntries()
	approvedTxEntries := []data.TxEntry{}
	if err != nil {
		log.WithFields(log.Fields{
			"function": "processQueue",
			"table":    "txlog",
		}).Fatal("Could not connect to database")
	}
	for _, j := range txEntries {
		person, err := ldapsearch.GetPerson(j.UniqueID)
		if err != nil {
			log.WithFields(log.Fields{
				"user":     j.UniqueID,
				"function": "processQueue",
			}).Warn("Ldap search failed")
		}
		if len(person.FirstName) == 0 && j.TxType == "add" {
			err = data.DeleteTxEntry(&j)
			if err != nil {
				log.WithFields(log.Fields{
					"uid":    j.UniqueID,
					"txtype": j.TxType,
				}).Warn("unable to remove TxEntry")
			}
			log.WithFields(log.Fields{
				"uid":    j.UniqueID,
				"txtype": j.TxType,
			}).Warn("Unable to lookup user in Ldap, removing from transaction log")
		} else {
			approvedTxEntries = append(approvedTxEntries, j)
		}
	}

	// break recursion if no more entries
	resultsReturned := len(approvedTxEntries)
	if resultsReturned < 1 {
		return
	} else {
		if resultsReturned < 6 {
			log.WithFields(log.Fields{
				"txEntries": approvedTxEntries,
			}).Info("Processing transactions")
		}
	}
	items := make([]umapi.Item, resultsReturned)
	for i, j := range approvedTxEntries {
		switch j.TxType {
		case "add":
			items[i] = umapi.GenAddItem(j.UniqueID, config.C.AdobeGroup)
		case "remove":
			items[i] = umapi.GenRemoveItem(j.UniqueID, config.C.AdobeGroup)
		default:
			log.WithFields(log.Fields{
				"function": "processQueue",
				"txtype":   j.TxType,
				"uid":      j.UniqueID,
			}).Fatal("Unknown txtype")
		}
	}
	requestBody, err := json.Marshal(items)
	if err != nil {
		log.Fatal("Unable to marshal requestBody")
	}
	var responseCode, retryAmount, numRequests int
	var actionResponse umapi.ActionResponse
	for responseCode != 200 {
		numRequests++
		response, err := umapi.SendRequest(string(requestBody), umapi.Token)
		if err != nil {
			log.WithFields(log.Fields{
				"request_length": len(requestBody),
				"token_type":     umapi.Token.TokenType,
				"token_exp":      umapi.Token.ExpiresIn,
			}).Fatal(err)
		}
		statusCodeString := strconv.Itoa(response.StatusCode)
		responsesTotal.With(prometheus.Labels{"status": statusCodeString}).Inc()
		switch response.StatusCode {
		case 200:
			defer response.Body.Close()
			output, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.WithFields(log.Fields{
					"code": response.StatusCode,
					"step": "load response body",
				}).Fatal(err)
			}
			err = json.Unmarshal(output, &actionResponse)
			if err != nil {
				log.WithFields(log.Fields{
					"code": response.StatusCode,
					"step": "unmarshal action response",
				}).Fatal(err)
			}
			responseCode = response.StatusCode
		case 400:
			log.WithFields(log.Fields{
				"request": "action",
				"code":    response.StatusCode,
			}).Fatal("Bad request or Service Account Integration Certificate has expired.")
		case 401:
			log.WithFields(log.Fields{
				"request": "action",
				"code":    response.StatusCode,
			}).Warn("Possible causes are invalid token, expired token or invalid organization.")
			umapi.Token.Renew()
		case 403:
			log.WithFields(log.Fields{
				"request": "action",
				"code":    response.StatusCode,
			}).Fatal("Missing API key or API key is not permitted access.")
		case 429:
			retryAfterString := response.Header.Get("Retry-After")
			retryAfterInt, err := strconv.Atoi(retryAfterString)
			if err != nil {
				log.WithFields(log.Fields{
					"request": "action",
					"code":    response.StatusCode,
				}).Fatal(err)
			}
			// add an additional second for good measure
			retryAmount += retryAfterInt + 1
			log.WithFields(log.Fields{
				"request": "action",
				"code":    response.StatusCode,
				"retry":   retryAmount,
			}).Warn("Too many requests")
			time.Sleep(time.Duration(retryAmount) * time.Second)
		default:
			log.WithFields(log.Fields{
				"request": "action",
				"code":    response.StatusCode,
			}).Fatal("Unhandled response code. Mudwork config may be incorrect.")
		}

	}
	if numRequests > 1 {
		log.WithFields(log.Fields{
			"num_requests": numRequests,
		}).Warn("Response code 200 required more than one request")
	}
	log.WithFields(log.Fields{
		"completed":           actionResponse.Completed,
		"notCompleted":        actionResponse.NotCompleted,
		"completedInTestMode": actionResponse.CompletedInTestMode,
	}).Info("Results")
	switch actionResponse.Result {
	case "success":
		// delete tx entries from txlog, add to users table
		for _, j := range approvedTxEntries {
			err = data.DeleteTxEntry(&j)
			if err != nil {
				log.WithFields(log.Fields{
					"table":  "txlog",
					"user":   j.UniqueID,
					"txtype": j.TxType,
				}).Fatal("Unable to delete row")
			}
			if *config.FlagTestMode {
				log.Info("Test mode enabled. Skipping Users table modifications.")
			} else {
				switch j.TxType {
				case "add":
					err = data.InsertUser(j.UniqueID)
					if err != nil {
						log.WithFields(log.Fields{
							"table": "users",
							"user":  j.UniqueID,
						}).Fatal("Unable to insert row")
					}
				case "remove":
					err = data.DeleteUser(j.UniqueID)
					if err != nil {
						log.WithFields(log.Fields{
							"table": "users",
							"user":  j.UniqueID,
						}).Fatal("Unable to delete row")
					}
				default:
					// fatal unexpected result
					log.WithFields(log.Fields{
						"object":   "TxType",
						"UniqueID": j.UniqueID,
						"TxType":   j.TxType,
					}).Fatal("Unexpected TxType value")
				}
			}

		}
	case "partial":
		// delete tx entries from txlog, add succeeded to users table
		// warn failed
		errorsMap := make(map[string]int)
		respErrors := []umapi.ActionResponseError{}
		if actionResponse.Errors != nil {
			respErrors = make([]umapi.ActionResponseError, len(*actionResponse.Errors))
			for i, j := range *actionResponse.Errors {
				respErrors[i] = j
				userName := respErrors[i].User //[0:strings.Index(respErrors[i].User, "@")]
				errorsMap[userName] = i
			}
		}
		warningsMap := make(map[string]int)
		respWarnings := []umapi.ActionResponseWarning{}
		if actionResponse.Warnings != nil {
			respWarnings = make([]umapi.ActionResponseWarning, len(*actionResponse.Warnings))
			for i, j := range *actionResponse.Warnings {
				respWarnings[i] = j
				userName := respWarnings[i].User //[0:strings.Index(respWarnings[i].User, "@")]
				warningsMap[userName] = i
			}
		}
		for _, j := range approvedTxEntries {
			jUniqueID := fmt.Sprintf("%s@%s", j.UniqueID, config.C.Enterprise["Domain"])
			err = data.DeleteTxEntry(&j)
			if err != nil {
				log.WithFields(log.Fields{
					"table":  "txlog",
					"user":   j.UniqueID,
					"txtype": j.TxType,
				}).Fatal("Unable to delete row")
			}
			if elem, ok := errorsMap[jUniqueID]; ok {
				log.WithFields(log.Fields{
					"error_code": respErrors[elem].ErrorCode,
					"user":       respErrors[elem].User,
					"message":    respErrors[elem].Message,
				}).Warn("Action failed")
				continue
			}
			if elem, ok := warningsMap[jUniqueID]; ok {
				log.WithFields(log.Fields{
					"error_code": respWarnings[elem].WarningCode,
					"user":       respWarnings[elem].User,
					"message":    respWarnings[elem].Message,
				}).Warn("Action returned warning")
			}
			if *config.FlagTestMode {
				log.Info("Test mode enabled. Skipping Users table modifications.")
			} else {
				switch j.TxType {
				case "add":
					err = data.InsertUser(j.UniqueID)
					if err != nil {
						log.WithFields(log.Fields{
							"table": "users",
							"user":  j.UniqueID,
						}).Fatal("Unable to insert row")
					}
				case "remove":
					err = data.DeleteUser(j.UniqueID)
					if err != nil {
						log.WithFields(log.Fields{
							"table": "users",
							"user":  j.UniqueID,
						}).Fatal("Unable to delete row")
					}
				default:
					// fatal unexpected result
					log.WithFields(log.Fields{
						"object":   "TxType",
						"UniqueID": j.UniqueID,
						"TxType":   j.TxType,
					}).Fatal("Unexpected TxType value")
				}
			}
		}

	case "error":
		// delete tx entries from txlog, warn failed
		for _, j := range approvedTxEntries {
			err = data.DeleteTxEntry(&j)
			if err != nil {
				log.WithFields(log.Fields{
					"table":  "txlog",
					"user":   j.UniqueID,
					"txtype": j.TxType,
				}).Fatal("Unable to delete row")
			}
		}
		for _, j := range *actionResponse.Errors {
			log.WithFields(log.Fields{
				"error_code": j.ErrorCode,
				"user":       j.User,
				"message":    j.Message,
			}).Warn("Action failed")
		}
	default:
		// fatal unexpected result
		log.WithFields(log.Fields{
			"object": "ActionResponse",
			"result": actionResponse.Result,
		}).Fatal("Unexpected result value")
	}
	// check for more entries
	processQueue()
}
