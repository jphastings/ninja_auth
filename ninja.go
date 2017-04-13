package main

import (
  "crypto/rand"
  "encoding/base64"
  "io/ioutil"
  "fmt"
  "log"
  "os"
  "strings"
  "net/http"
  "encoding/json"

  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"

  "github.com/gin-contrib/sessions"
  "github.com/gin-gonic/gin"
  "github.com/jphastings/ninja_auth/lib/multiproxy"
)

type User struct {
  // Attributes of the Google response which we're interested in
  Email string `json:"email"`
  HostedDomain string `json:"hd"`
}

var conf *oauth2.Config
var state string
var store = sessions.NewCookieStore([]byte(os.Getenv("NINJA_SECRET")))
var acceptable_domains = strings.Split(os.Getenv("NINJA_ACCEPTABLE_DOMAINS"), ",")
var proxy = multiproxy.NewMultiProtocolSingleHostReverseProxy(fmt.Sprintf("127.0.0.1:%s", os.Getenv("NINJA_PROXY_PORT")))

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
    log.Printf("[Ninja] Session state code doesn't match response from Google, redirecting for re-auth.")
    c.Redirect(302, "/")
    return
  }

  tok, err := conf.Exchange(oauth2.NoContext, c.Query("code"))
  if err != nil {
    log.Printf("[Ninja] Unable to exchange for token, redirecting for re-auth.")
    c.Redirect(302, "/")
    return
  }

  client := conf.Client(oauth2.NoContext, tok)
  email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
  if err != nil {
    log.Printf("[Ninja] User information unavailable from Google: %s", err)
    c.AbortWithError(http.StatusBadRequest, err)
    return
  }
  defer email.Body.Close()

  jsonBlob, _ := ioutil.ReadAll(email.Body)
  authedUser := User{};
  err = json.Unmarshal(jsonBlob, &authedUser)
  session.Set("email", authedUser.Email)
  session.Set("hosted_domain", authedUser.HostedDomain)
  session.Save()

  if isAuthorized(session) {
    log.Printf("[Ninja] User %s (of %s) authenticated successfully.", authedUser.Email, authedUser.HostedDomain)
    originalPath := session.Get("path").(string)
    c.Redirect(302, originalPath)
  } else {
    log.Printf("[Ninja] User %s (of %s) authenticated but forbidden.", authedUser.Email, authedUser.HostedDomain)
    c.Status(http.StatusForbidden)
    c.Writer.Write([]byte("<h1>Soz</h1>"))
  }
  return
}

func proxyHandler(c *gin.Context) {
  session := sessions.Default(c)

  if isAuthorized(session) {
    log.Printf("[Ninja] Authorized request from %s being forwarded downstream.", session.Get("email"))
    proxy.ServeHTTP(c.Writer, c.Request)
  } else {
    log.Printf("[Ninja] Unuauthorized request, redirecting through auth process.")

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

  port := os.Getenv("PORT")
  log.Printf("[Ninja] Starting NinjaAuth reverse proxy service on port %s", port)
  router.Run(fmt.Sprintf(":%s", port))
}
