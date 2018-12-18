package ldapsearch
  
import (
        "fmt"
        "github.com/cosmouser/mudwork/config"
        "gopkg.in/ldap.v2"
)

type Person struct {
        FirstName string
        LastName  string
        Email     string
        Uid       string
}

func GetPerson(uid string) (*Person, error) {
        l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.C.LdapUrl, config.C.LdapPort))
        if err != nil {
                return nil, err
        }
        defer l.Close()
        searchRequest := ldap.NewSearchRequest(
                config.C.LdapBase,
                ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
                fmt.Sprintf("(&(uid=%s))", uid),
                []string{config.C.LdapFirstName, config.C.LdapLastName},
                nil,
        )
        sr, err := l.Search(searchRequest)
        if err != nil {
                return nil, err
        }
        p := &Person{}
        for _, entry := range sr.Entries {
                p.FirstName = entry.GetAttributeValue(config.C.LdapFirstName)
                p.LastName = entry.GetAttributeValue(config.C.LdapLastName)
        }
        p.Email = fmt.Sprintf("%s@%s", uid, config.C.Enterprise["Domain"])
        p.Uid = uid
        return p, nil
}
