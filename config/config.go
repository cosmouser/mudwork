package config

import (
        "flag"
        "github.com/BurntSushi/toml"
        log "github.com/sirupsen/logrus"
)

type Config struct {
        JssUrl        string
        JssIP         string
        ApiUser       string
        ApiPass       string
        AdvSearchID   int
        CirrupUser    string
        DbPath        string
        LdapFirstName string
        LdapLastName  string
        LdapUrl       string
        LdapPort      int
        LdapBase      string
        AdobeGroup    string
        Server        map[string]string
        Enterprise    map[string]string
}

// Server map
//      Host           string
//      Endpoint       string
//      ImsHost        string
//      ImsEndpointJwt string

// Enterprise map
//      Domain         string
//      OrgID          string
//      APIKey         string
//      ClientSecret   string
//      TechAcctstring string
//      PrivKeyPath    string

var C Config
var FlagGroups *bool
var FlagProd *bool
var FlagNoInit *bool
var FlagTestMode *bool
var FlagPort *int

func init() {
        var err error
        configPath := flag.String("config", "./config/mudwork.toml", "use -config to specify the config file to load")
        FlagGroups = flag.Bool("groups", false, "query Adobe for a list of each group, print then quit")
        FlagNoInit = flag.Bool("noinit", false, "set -noinit if a token does not need to be initialized")
        FlagTestMode = flag.Bool("testmode", false, "Sends Adobe requests in test mode")
        FlagProd = flag.Bool("prod", false, "set -prod for persistent storage / production server")
        FlagPort = flag.Int("p", 8443, "sets the port number for mudwork to listen on")
        flag.Parse()
        if *configPath == "" {
                log.Fatal("Could not load config. Please use -config to specify a config file")
        }
        _, err = toml.DecodeFile(*configPath, &C)
        if err != nil {
                panic(err)
        }
}
