package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/sessions"

	_ "golang.org/x/net/context"
	"golang.org/x/oauth2"

)

// Authentication + Encryption key pairs
var sessionStoreKeyPairs = [][]byte{
	[]byte("something-very-secret"),
	nil,
}

var store sessions.Store

var (
	clientID string
	config   *oauth2.Config
)

type User struct {
	Email       string
	DisplayName string
}

func init() {
	// Create file system store with no size limit
	fsStore := sessions.NewFilesystemStore("", sessionStoreKeyPairs...)
	fsStore.MaxLength(0)
	store = fsStore

	gob.Register(&User{})
	gob.Register(&oauth2.Token{})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	clientID = GetEnv("CLIENT_ID","")
	if clientID == "" {
		log.Fatal("CLIENT_ID must be set.")
	}

	secret := GetEnv("CLIENT_SECRET","") // no client secret by default
	authUrl := GetEnv("AUTH_URL","https://login.microsoftonline.com/common/oauth2/authorize")
	tokenUrl := GetEnv("TOKEN_URL","https://login.microsoftonline.com/common/oauth2/token")
	scopes := GetEnv("OIDC_SCOPES","User.Read")
	redirectURI := GetEnv("REDIRECT_URL","http://localhost:8080/seldon-deploy/auth/callback")

	config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		RedirectURL:  redirectURI,

		Endpoint: oauth2.Endpoint{
			AuthURL:  authUrl,
			TokenURL: tokenUrl,
		},

		Scopes: []string{scopes},
	}

	callbackUri := GetEnv("CALLBACK_PATH","/seldon-deploy/auth/callback")
	http.Handle("/seldon-deploy", handle(IndexHandler))
	http.Handle(callbackUri, handle(CallbackHandler))

	port := GetEnv("PORT","8000")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		log.Println("ENV VAR DEFAULTED "+key+": "+defaultValue)
		return defaultValue
	}
	log.Println("ENV VAR "+key+": "+value)
	return value
}

type handle func(w http.ResponseWriter, req *http.Request) error

func (h handle) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Handler panic: %v", r)
		}
	}()
	if err := h(w, req); err != nil {
		log.Printf("Handler error: %v", err)

		if httpErr, ok := err.(Error); ok {
			http.Error(w, httpErr.Message, httpErr.Code)
		}
	}
}

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	if e.Message == "" {
		e.Message = http.StatusText(e.Code)
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func IndexHandler(w http.ResponseWriter, req *http.Request) error {
	session, _ := store.Get(req, "session")

	var token *oauth2.Token
	if req.FormValue("logout") != "" {
		session.Values["token"] = nil
		sessions.Save(req, w)
	} else {
		if v, ok := session.Values["token"]; ok {
			if v!= nil {
				token = v.(*oauth2.Token)
			}
		}
	}

	resourceURI := GetEnv("RESOURCE_URI","")
	resourceURIParam := oauth2.SetAuthURLParam("resource", resourceURI)

	var data = struct {
		Token   *oauth2.Token
		AuthURL string
	}{
		Token:   token,
		AuthURL: config.AuthCodeURL(SessionState(session), oauth2.AccessTypeOnline, resourceURIParam),
	}

	return indexTempl.Execute(w, &data)
}

var indexTempl = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
  <head>
    <title>OAuth2 Test Tool</title>

    <link href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  </head>
  <body class="container-fluid">
    <div class="row">
      <div class="col-xs-4 col-xs-offset-4">
        <h1>OAuth2 Test Tool</h1>
{{with .Token}}
        <div id="displayName"></div>
		<div style="white-space: nowrap">{{.AccessToken}}</div>
        <a href="/seldon-deploy?logout=true">Logout</a>
{{else}}
        <a href="{{$.AuthURL}}">Login</a>
{{end}}
      </div>
    </div>

    <script src="https://code.jquery.com/jquery-3.2.1.min.js" integrity="sha256-hwg4gsxgFZhOsEEamdOYGBf13FyQuiTwlAQgxVSNgt4=" crossorigin="anonymous"></script>
    <script>
{{with .Token}}
      var token = {{.}};

      $.ajax({
        url: 'https://graph.windows.net/me?api-version=1.6',
        dataType: 'json',
        success: function(data, status) {
        	$('#displayName').text('Welcome ' + data.displayName);
        },
        beforeSend: function(xhr, settings) {
          xhr.setRequestHeader('Authorization', 'Bearer ' + token.access_token);
        }
      });
{{end}}
    </script>
  </body>
</html>
`))

func CallbackHandler(w http.ResponseWriter, req *http.Request) error {
	session, _ := store.Get(req, "session")

	if req.FormValue("state") != SessionState(session) {
		return Error{http.StatusBadRequest, "invalid callback state"}
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", req.FormValue("code"))
	form.Set("redirect_uri",GetEnv("REDIRECT_URL","http://localhost:8000/seldon-deploy/callback"))
	form.Set("resource", GetEnv("RESOURCE_URI","https://graph.windows.net"))
	form.Set("client_secret",GetEnv("CLIENT_SECRET",""))
	tokenReq, err := http.NewRequest(http.MethodPost, config.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("error creating token request: %v", err)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("error performing token request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("token response was %s", resp.Status)
	}

	log.Println("raw token")
	var data string
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	data = string(bodyBytes)
	spew.Dump(data)

	var token oauth2.Token
	if err := json.Unmarshal(bodyBytes,&token); err != nil {
		return fmt.Errorf("error decoding JSON response: %v", err)
	}
	log.Println("decoded token")
	spew.Dump(token)



	session.Values["token"] = &token
	if err := sessions.Save(req, w); err != nil {
		return fmt.Errorf("error saving session: %v", err)
	}

	http.Redirect(w, req, "/seldon-deploy", http.StatusFound)
	return nil
}

func SessionState(session *sessions.Session) string {
	return base64.StdEncoding.EncodeToString(sha256.New().Sum([]byte(session.ID)))
}

func dump(v interface{}) {
	spew.Dump(v)
}
