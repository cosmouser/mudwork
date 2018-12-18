package umapi

import (
	"encoding/json"
	"fmt"
	"github.com/cosmouser/mudwork/config"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// SendRequest makes an rpc to Adobe's umapi
func SendRequest(body string, token *AccessResponse) (*http.Response, error) {
	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	bodyReader := strings.NewReader(body)
	resourceURI := fmt.Sprintf("https://%s%s/action/%s",
		config.C.Server["Host"],
		config.C.Server["Endpoint"],
		config.C.Enterprise["OrgID"],
	)
	if *config.FlagTestMode {
		resourceURI += "?testOnly=true"
		log.Info("testOnly set to true")
	}
	req, err := http.NewRequest("POST", resourceURI, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("x-api-key", config.C.Enterprise["APIKey"])
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetGroups returns the Groups from the configured Adobe endpoint
func GetGroups(token *AccessResponse) (groups []Group, err error) {
	var lastPage bool
	var numRenews, retryAmount int
	for i := 0; lastPage != true; i++ {
		gR, err := getGroupPage(i, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		switch gR.StatusCode {
		case 200:
			defer gR.Body.Close()
			output, err := ioutil.ReadAll(gR.Body)
			if err != nil {
				return nil, err
			}
			gro := &GroupResponse{}
			err = json.Unmarshal(output, gro)
			if err != nil {
				return nil, err
			}
			for _, j := range gro.Groups {
				groups = append(groups, j)
			}
			lastPage = gro.LastPage
		case 400:
			//			err = fmt.Errorf("GetGroups received response code 400. Some parameters of the request were not understood by the server or the Service Account Integration certificate has expired.")
			//			return nil, err
			log.WithFields(log.Fields{
				"request": "GetGroups",
				"code":    gR.StatusCode,
				"page":    i,
			}).Fatal("Bad request or Service Account Integration Certificate has expired.")
		case 401:
			log.WithFields(log.Fields{
				"request": "GetGroups",
				"code":    gR.StatusCode,
				"page":    i,
			}).Warn("Possible causes are invalid token, expired token or invalid organization.")
			token.Renew()
			numRenews++
			i--
		case 403:
			log.WithFields(log.Fields{
				"request": "GetGroups",
				"code":    gR.StatusCode,
				"page":    i,
			}).Fatal("Missing API key or API key is not permitted access.")
		case 429:
			retryAfterString := gR.Header.Get("Retry-After")
			retryAfterInt, err := strconv.Atoi(retryAfterString)
			if err != nil {
				panic(err)
			}
			retryAmount += retryAfterInt + 1 // add an additional second for good measure
			log.WithFields(log.Fields{
				"request": "GetGroups",
				"code":    gR.StatusCode,
				"page":    i,
				"retry":   retryAmount,
			}).Warn("Too many requests")
			time.Sleep(time.Duration(retryAmount) * time.Second)
			i--
		default:
			log.WithFields(log.Fields{
				"request": "GetGroups",
				"code":    gR.StatusCode,
				"page":    i,
			}).Fatal("Unhandled response code. Mudwork config may be incorrect.")
		}
	}
	return groups, err
}

// getGroupPage returns a single page from the groups endpoint
func getGroupPage(page int, token *AccessResponse) (*http.Response, error) {
	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	resourceURI := fmt.Sprintf("https://%s%s/groups/%s/%d",
		config.C.Server["Host"],
		config.C.Server["Endpoint"],
		config.C.Enterprise["OrgID"],
		page,
	)
	req, err := http.NewRequest("GET", resourceURI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("x-api-key", config.C.Enterprise["APIKey"])
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Renew renews the token
func (token *AccessResponse) Renew() {
	var err error
	log.Info("Renewing Token")
	generated_jwt := GenerateJwt()
	accessRequest := AccessRequestBody(generated_jwt)
	newToken, err := RequestAccess(accessRequest)
	if err != nil {
		log.Printf("%s\n", err)
	}
	token.TokenType = newToken.TokenType
	token.AccessToken = newToken.AccessToken
	token.ExpiresIn = newToken.ExpiresIn
}

var Token *AccessResponse

func init() {
	Token = &AccessResponse{}
}
