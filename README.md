# mudwork
Mudwork is a web server that synchronizes Adobe named user software licenses with the membership of a [Cirrup](https://github.com/cosmouser/cirrup) managed Jamf Pro computer group.

## How Mudwork Functions
Mudwork uses an event-response design. The event is Cirrup making a change to the membership of a dynamic Static Computer Group in the Jamf Pro JSS. The response is that Mudwork uses one of the Advanced Computer Searches in the Jamf Pro JSS to see if a computer was added with a username that isn’t assigned to any of the other computers in the list or if a username that used to be assigned to one of the computers in the list is no longer assigned to any of the computers. If one of these cases is true, that is, if a new username has been added or an existing one no longer shows up, then Mudwork sends a request to Adobe’s User Management API endpoint to add the new user or remove the old user. 

When adding a new user, Mudwork provisions a new Federated ID for the user using the givenName and sn fields from a directory query that it sends directly to the campus directory server. Then Mudwork assigns the software entitlement specified in its configuration file to the newly provisioned Federated ID. Here, the Federated ID has the user’s email address as the username and their CruzID Gold password as the password. 

When removing a user that no longer shows up as assigned to any of the computers in the dynamic Static Computer Group, Mudwork merely removes the software specified in its configuration file from the user’s Federated ID.

## System Details
In order to operate successfully and securely, Mudwork runs behind an HTTPS reverse proxy server such as Nginx or Apache httpd. The host running the reverse proxy server should be able to receive incoming traffic on port 443 from the Jamf Pro JSS and be able to send TCP traffic to the campus directory server, Jamf Pro JSS and Adobe User Management API host. A self signed certificate needs to be generated for signing the JWT’s that Adobe’s API requires for authenticating requests against its endpoints. This certificate is not used for anything other than verifying that the public key and private key match. 

Since Mudwork is a statically-linked binary executable that uses SQLite for a database, it has no external dependencies and does not require a JVM and or MySQL/NoSQL database to be set up on its host before it will work. The user account that runs the Mudwork process will need to be able to read and write to the directory that the configuration file specifies its database file should reside and needs read access to the JWT private key and configuration file.

Mudwork requires an operational Cirrup installation before it can start to manage licenses. Mudwork also requires a domain with Federated ID’s setup and operational through an identity provider such as Shibboleth or Okta.

## Deployment
Before beginning a deployment of Mudwork, create a folder to store the configuration file, database file, keys, certificates and a user with read write permission to the folder. 
1. Create a User Management API integration at https://console.adobe.io/ with a self signed certificate. Keep track of the certificate’s expiration date as the integration will need to be renewed by the time the certificate expires.
2. Copy the Mudwork binary onto the host and fill out each field of the configuration file except for the AdobeGroup, AdvSearchID, ApiUser and ApiPass fields. 
3. Run mudwork -config /path/to/config.txt -groups
4. If your configuration file has been successfully filled out and your Adobe User Management API integration are properly configured then you will see a list of product entitlements for your institution. Find the group that corresponds to the product you want to manage with Mudwork and then fill it in as the value for the AdobeGroup field.
5. Create a user in the Jamf Pro JSS for Mudwork to use. The only privilege that Mudwork’s JSS user needs is READ access to Advanced Computer Searches, in the Jamf Pro Server Objects section. Fill in the ApiUser and ApiPass fields with this user’s credentials.
6. Create an Advanced Computer Search in the Jamf Pro JSS that displays all of the computers in the dynamic Static Computer Groups that Cirrup manages on your Jamf Pro JSS. For the Display section of the Advanced Computer Search, leave all of the boxes unchecked except for Username in the User and Location section.
7. Find the ID of the Advanced Computer Search that you made by looking at the id parameter in the URL when looking at the search in your web browser. Put this number as the value for the AdvSearchID field in the configuration file.
8. Create a proxy rule for the Mudwork process in your webserver (httpd, nginx, etc).
9. Create a systemd service file for your process, reload systemd, then start and enable your service.
10. Create a webhook in the Jamf Pro JSS that sends notifications to Mudwork when RestAPIOperations occur to the JSS.

## Configuration File
Mudwork uses Tom's Obvious, Minimal Language for its config file. Required files are below.
```
JssUrl          = "https://jss.uni.edu:8443"
JssIP           = "10.20.30.40" # IP of your JSS
ApiUser         = "mudwork jss user name goes here"
ApiPass         = "mudwork jss user password goes here"
AdvSearchID     = 26 # ID of adv search to find computers managed by Cirrup
CirrupUser      = "name of JSS account used by Cirrup"
DbPath          = "/path/to/mudwork_cache.db"
LdapFirstName   = "ldap attribute for first name goes here"
LdapLastName    = "ldap attribute for last name goes here"
LdapUrl         = "ldap host FQDN goes here"
LdapPort        = 389
LdapBase        = "ldap search base goes here"
AdobeGroup      = "Adobe Product Group goes here"

[Server]
Host            = "usermanagement.adobe.io"
Endpoint        = "/v2/usermanagement"
ImsHost         = "ims-na1.adobelogin.com"
ImsEndpointJwt  = "/ims/exchange/jwt"

[Enterprise]
Domain          = "uni.edu"
OrgID           = "orgid goes here@AdobeOrg"
APIKey          = "insert api key here"
ClientSecret    = "client secret goes here"
TechAcct        = "tech acct goes here @techacct.adobe.com"
PrivKeyPath     = "/path/to/private.key"
```

## Jamf Pro JSS Webhook Configuration
After everything is set up and running, a webhook must be configured in the Jamf Pro JSS for sending notifications to Mudwork when Cirrup makes a change. The path in the Webhook URL corresponds to the path that your web server has for forwarding traffic to mudwork.
