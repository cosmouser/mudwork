package umapi

import (
	"encoding/json"
	"fmt"
	"github.com/cosmouser/mudwork/config"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// AccessResponse is the json body of the response from
// RequestAccess. It contains the AccessToken that is used
// for authorizing User Management API requests.
type AccessResponse struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GenerateJwt uses the information from the config to create a
// jwt that can be put into AccessRequestBody
func GenerateJwt() string {
	signBytes, err := ioutil.ReadFile(config.C.Enterprise["PrivKeyPath"])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	mySigningKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	aud := fmt.Sprintf("https://%s/c/%s", config.C.Server["ImsHost"], config.C.Enterprise["APIKey"])
	dur := time.Second * 60 * 60 * 24
	exp := time.Now().Add(dur).Unix()

	type MyCustomClaims struct {
		EntUserSdk bool `json:"https://ims-na1.adobelogin.com/s/ent_user_sdk"`
		jwt.StandardClaims
	}
	claims := MyCustomClaims{
		true,
		jwt.StandardClaims{
			Audience:  aud,
			ExpiresAt: exp,
			Issuer:    config.C.Enterprise["OrgID"],
			Subject:   config.C.Enterprise["TechAcct"],
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	return ss
}

// AccessRequestBody takes a signed jwt and generates the body
// for putting into RequestAccess for returning the AccessToken
func AccessRequestBody(jsonToken string) string {
	vals := url.Values{}
	vals.Set("client_id", config.C.Enterprise["APIKey"])
	vals.Set("client_secret", config.C.Enterprise["ClientSecret"])
	vals.Set("jwt_token", jsonToken)
	return vals.Encode()
}

// RequestAccess sends a request to Adobe's IMS and returns
// an *AccessResponse whose AccessToken authorizes requests made
// to Adobe's User Management API
func RequestAccess(body string) (*AccessResponse, error) {
	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	bodyReader := strings.NewReader(body)
	resourceURI := fmt.Sprintf("https://%s%s", config.C.Server["ImsHost"], config.C.Server["ImsEndpointJwt"])
	req, err := http.NewRequest("POST", resourceURI, bodyReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Cache-Control", "no-cache")
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return nil, err
	}
	defer resp.Body.Close()
	output, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return nil, err
	}
	var accResp AccessResponse
	err = json.Unmarshal(output, &accResp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return nil, err
	}
	return &accResp, nil
}
