package basic

import (
	"fmt"
	"net/http"

	"github.com/janekolszak/idp/core"
	"golang.org/x/crypto/bcrypt"
)

// Basic Authentication checker.
// Expects Storage to return plain text passwords
type BasicAuth struct {
	Htpasswd Htpasswd
	Realm    string
}

func NewBasicAuth(htpasswdFileName string, realm string) (*BasicAuth, error) {
	b := new(BasicAuth)

	err := b.Htpasswd.Load(htpasswdFileName)
	if err != nil {
		return nil, err
	}

	b.Realm = realm

	return b, nil
}

func (c *BasicAuth) Check(r *http.Request) (user string, err error) {

	// TODO: Pre-validate user and password

	user, pass, ok := r.BasicAuth()
	if !ok {
		user = ""
		err = core.ErrorAuthenticationFailure
		return
	}

	hash, err := c.Htpasswd.Get(user)
	if err != nil {
		// Prevent timing attack
		bcrypt.CompareHashAndPassword([]byte{}, []byte(pass))
		user = ""
		err = core.ErrorAuthenticationFailure
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	if err != nil {
		user = ""
		err = core.ErrorAuthenticationFailure
	}

	return
}

func (c *BasicAuth) Register(r *http.Request) (user string, err error) {
	err = core.ErrorNotImplemented
	return
}

func (c *BasicAuth) WriteError(w http.ResponseWriter, r *http.Request, _ error) error {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, c.Realm))
	http.Error(w, "authorization failed", http.StatusUnauthorized)
	return nil
}

func (c *BasicAuth) Write(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (c *BasicAuth) WriteRegister(w http.ResponseWriter, r *http.Request) error {
	return core.ErrorNotImplemented
}

func (c *BasicAuth) Verify(r *http.Request) (string, error) {
	return "", core.ErrorNotImplemented
}

func (c *BasicAuth) WriteVerify(w http.ResponseWriter, r *http.Request, userid string) error {
	return core.ErrorNotImplemented
}
