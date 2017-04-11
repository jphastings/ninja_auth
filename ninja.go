package main

import (
  "crypto/rand"
  "encoding/base64"
  "io/ioutil"
  "fmt"
  "os"
  "strings"
  "net/url"
  "net/http"
  "net/http/httputil"
  "encoding/json"

  "github.com/gin-contrib/sessions"
  "github.com/gin-gonic/gin"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
)

type User struct {
  Sub string `json:"sub"`
  Name string `json:"name"`
  GivenName string `json:"given_name"`
  FamilyName string `json:"family_name"`
  Profile string `json:"profile"`
  Picture string `json:"picture"`
  Email string `json:"email"`
  EmailVerified string `json:"email_verified"`
  Gender string `json:"gender"`
  HostedDomain string `json:"hd"`
}

var conf *oauth2.Config
var state string
var store = sessions.NewCookieStore([]byte(os.Getenv("NINJA_SECRET")))
var acceptable_domains = strings.Split(os.Getenv("NINJA_ACCEPTABLE_DOMAINS"), ",")
var proxyHostUrl = url.URL{
  Scheme: "http",
  Host: fmt.Sprintf("127.0.0.1:%s", os.Getenv("NINJA_PROXY_PORT")),
}
var proxy = httputil.NewSingleHostReverseProxy(&proxyHostUrl)

func randToken() string {
  b := make([]byte, 32)
  rand.Read(b)
  return base64.StdEncoding.EncodeToString(b)
}

func init() {
  gin.SetMode(gin.ReleaseMode)
  conf = &oauth2.Config{
    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
    RedirectURL:  os.Getenv("NINJA_BASE_URL") + "/ninja_auth",
    Scopes: []string{
        "https://www.googleapis.com/auth/userinfo.email",
    },
    Endpoint: google.Endpoint,
  }
}

func getLoginURL(state string) string {
  return conf.AuthCodeURL(state)
}

func isAuthorized(session sessions.Session) bool {
  hd := fmt.Sprintf("%v", session.Get("hosted_domain"))

  for _, domain := range acceptable_domains {
    if domain == hd {
      return true
    }
  }
  return false
}

func authHandler(c *gin.Context) {
  session := sessions.Default(c)

  retrievedState := session.Get("state")
  if retrievedState != c.Query("state") {
    c.Redirect(302, "/")
    return
  }

  tok, err := conf.Exchange(oauth2.NoContext, c.Query("code"))
  if err != nil {
    // Issue with the code, try re-authing
    c.Redirect(302, "/")
    return
  }

  client := conf.Client(oauth2.NoContext, tok)
  email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
  if err != nil {
    c.AbortWithError(http.StatusBadRequest, err)
    return
  }
  defer email.Body.Close()

  jsonBlob, _ := ioutil.ReadAll(email.Body)
  authedUser := User{};
  err = json.Unmarshal(jsonBlob, &authedUser)
  session.Set("hosted_domain", authedUser.HostedDomain)
  session.Save()

  if isAuthorized(session) {
    originalPath := session.Get("path").(string)
    c.Redirect(302, originalPath)
  } else {
    c.Status(http.StatusForbidden)
    c.Writer.Write([]byte("<h1>Soz</h1>"))
  }
  return
}

func proxyHandler(c *gin.Context) {
  session := sessions.Default(c)

  if isAuthorized(session) {
    proxy.ServeHTTP(c.Writer, c.Request)
  } else {
    state = randToken()
    session.Set("state", state)
    session.Set("path", c.Request.URL.String())
    session.Save()
    c.Redirect(302, getLoginURL(state))
  }
  return
}

func main() {
  router := gin.Default()
  router.Use(sessions.Sessions("ninja", store))

  router.GET("/ninja_auth", authHandler)
  router.NoRoute(proxyHandler)

  router.Run(fmt.Sprintf(":%s", os.Getenv("PORT")))
}
