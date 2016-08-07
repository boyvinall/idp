package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/janekolszak/idp/core"
	"github.com/janekolszak/idp/helpers"
	"github.com/janekolszak/idp/providers/cookie"
	"github.com/janekolszak/idp/providers/form"
	"github.com/janekolszak/idp/userdb/memory"
	"github.com/julienschmidt/httprouter"

	_ "github.com/mattn/go-sqlite3"
)

const (
	consent = `<html>
<head></head>
<body>
<p>User:        {{.User}} </p>
<p>Client Name: {{.Client.Name}} </p>
<p>Scopes:      {{range .Scopes}} {{.}} {{end}} </p>
<p>Do you agree to grant access to those scopes? </p>
<p><form method="post">
	<input type="submit" name="answer" value="y">
	<input type="submit" name="answer" value="n">
</form></p>
</body></html>
`

	loginform = `
<html>
<head></head>
<body>
<form method="post">
	<p>Example App</p>
	<p>username <input type="text" name="username"></p>
	<p>password <input type="password" name="password" autocomplete="off"></p>
	<input type="submit">
	<a href="{{.RegisterURI}}">Register</a>
</form>
<hr>
{{.Msg}}
<body>
</html>
`
)

var (
	hydraURL     = flag.String("hydra", "https://hydra:4444", "Hydra's URL")
	configPath   = flag.String("conf", ".hydra.yml", "Path to Hydra's configuration")
	htpasswdPath = flag.String("htpasswd", "/etc/idp/htpasswd", "Path to credentials in htpasswd format")
	cookieDBPath = flag.String("cookie-db", "/etc/idp/remember.db3", "Path to a database with remember me cookies")
	staticFiles  = flag.String("static", "", "directory to serve as /static (for CSS/JS/images etc)")
)

func main() {
	fmt.Println("Identity Provider started!")

	flag.Parse()
	// Read the configuration file
	hydraConfig := helpers.NewHydraConfig(*configPath)

	// Setup the providers
	userdb, err := memory.NewMemStore()
	if err != nil {
		panic(err)
	}

	err = userdb.LoadHtpasswd(*htpasswdPath)
	if err != nil {
		panic(err)
	}

	provider, err := form.NewFormAuth(form.Config{
		LoginForm:          loginform,
		LoginUsernameField: "username",
		LoginPasswordField: "password",

		// Store for
		UserStore:    userdb,
		UserVerifier: nil,

		// Validation options:
		Username: form.Complexity{
			MinLength: 1,
			MaxLength: 100,
			Patterns:  []string{".*"},
		},
		Password: form.Complexity{
			MinLength: 1,
			MaxLength: 100,
			Patterns:  []string{".*"},
		},
	})
	if err != nil {
		panic(err)
	}

	dbCookieStore, err := cookie.NewDBStore("sqlite3", *cookieDBPath)
	if err != nil {
		panic(err)
	}

	cookieProvider := &cookie.CookieAuth{
		Store:  dbCookieStore,
		MaxAge: time.Minute * 1,
	}

	idp := core.NewIDP(&core.IDPConfig{
		ClusterURL:            *hydraURL,
		ClientID:              hydraConfig.ClientID,
		ClientSecret:          hydraConfig.ClientSecret,
		KeyCacheExpiration:    10 * time.Minute,
		ClientCacheExpiration: 10 * time.Minute,
		CacheCleanupInterval:  30 * time.Second,

		// TODO: [IMPORTANT] Don't use CookieStore here
		ChallengeStore: sessions.NewCookieStore([]byte("something-very-secret")),
	})

	// Connect with Hydra
	err = idp.Connect()
	if err != nil {
		panic(err)
	}

	handler, err := CreateHandler(HandlerConfig{
		IDP:            idp,
		Provider:       provider,
		CookieProvider: cookieProvider,
		ConsentForm:    consent,
		StaticFiles:    *staticFiles,
	})

	router := httprouter.New()
	handler.Attach(router)
	http.ListenAndServe(":3000", router)

	idp.Close()
}
