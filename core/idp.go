package core

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/sessions"
	hclient "github.com/ory-am/hydra/client"
	hjwk "github.com/ory-am/hydra/jwk"
	hoauth2 "github.com/ory-am/hydra/oauth2"
	hydra "github.com/ory-am/hydra/sdk"
	"github.com/patrickmn/go-cache"
)

const (
	VerifyPublicKey   = "VerifyPublic"
	ConsentPrivateKey = "ConsentPrivate"
	ClientInfo        = "ClientInfo"
)

var encryptionkey = "something-very-secret"

type IDPConfig struct {
	ClientID              string        `yaml:"client_id"`
	ClientSecret          string        `yaml:"client_secret"`
	ClusterURL            string        `yaml:"hydra_address"`
	KeyCacheExpiration    time.Duration `yaml:"key_cache_expiration"`
	ClientCacheExpiration time.Duration `yaml:"client_cache_expiration"`
	CacheCleanupInterval  time.Duration `yaml:"cache_cleanup_interval"`
	ChallengeStore        sessions.Store
}

type IDP struct {
	config *IDPConfig

	// Communication with Hydra
	hc *hydra.Client

	// Http client for communicating with Hydra
	client *http.Client

	// Cache for all private and public keys
	cache *cache.Cache

	// Prepared cookie options for creating and deleting cookies
	// TODO: Is this the best way to do this?
	createChallengeCookieOptions *sessions.Options
	deleteChallengeCookieOptions *sessions.Options
}

func NewIDP(config *IDPConfig) *IDP {
	var idp = new(IDP)
	idp.config = config

	// TODO: Pass TTL and refresh period from config
	idp.cache = cache.New(config.KeyCacheExpiration, config.CacheCleanupInterval)
	idp.cache.OnEvicted(func(key string, value interface{}) { idp.refreshCache(key) })

	idp.createChallengeCookieOptions = new(sessions.Options)
	idp.createChallengeCookieOptions.Path = "/"      // TODO: More specific?
	idp.createChallengeCookieOptions.MaxAge = 60 * 5 // 5min
	idp.createChallengeCookieOptions.Secure = false  // TODO: Change to true
	idp.createChallengeCookieOptions.HttpOnly = false

	idp.deleteChallengeCookieOptions = new(sessions.Options)
	idp.deleteChallengeCookieOptions.Path = "/"     // TODO: More specific?
	idp.deleteChallengeCookieOptions.MaxAge = -1    // Mark for deletion
	idp.deleteChallengeCookieOptions.Secure = false // TODO: Change to true
	idp.deleteChallengeCookieOptions.HttpOnly = false

	return idp
}

// Called when any key expires
func (idp *IDP) refreshCache(key string) {
	switch key {
	case VerifyPublicKey:
		verifyKey, err := idp.getVerificationKey()
		if err != nil {
			return
		}
		idp.cache.Set(VerifyPublicKey, verifyKey, cache.DefaultExpiration)
		return

	case ConsentPrivateKey:
		consentKey, err := idp.getConsentKey()
		if err != nil {
			return
		}
		idp.cache.Set(ConsentPrivateKey, consentKey, cache.DefaultExpiration)
		return

	case ClientInfo:

		clients, err := idp.hc.Client.GetClients()
		if err != nil {
			return
		}
		idp.cache.Set(ClientInfo, clients, idp.config.ClientCacheExpiration)

		return

	default:
		return
	}
}

// Downloads the hydra's public key
func (idp *IDP) getVerificationKey() (*rsa.PublicKey, error) {

	jwk, err := idp.hc.JWK.GetKey(hoauth2.ConsentChallengeKey, "public")
	if err != nil {
		return nil, err
	}

	rsaKey, ok := hjwk.First(jwk.Keys).Key.(*rsa.PublicKey)
	if !ok {
		return nil, ErrorBadPublicKey
	}

	return rsaKey, nil
}

// Downloads the private key used for signing the consent
func (idp *IDP) getConsentKey() (*rsa.PrivateKey, error) {
	jwk, err := idp.hc.JWK.GetKey(hoauth2.ConsentEndpointKey, "private")
	if err != nil {
		return nil, err
	}

	rsaKey, ok := hjwk.First(jwk.Keys).Key.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrorBadPrivateKey
	}

	return rsaKey, nil
}

func (idp *IDP) Connect() error {
	var err error
	idp.hc, err = hydra.Connect(
		hydra.ClientID(idp.config.ClientID),
		hydra.ClientSecret(idp.config.ClientSecret),
		hydra.ClusterURL(idp.config.ClusterURL),
		hydra.SkipTLSVerify(),
	)

	if err != nil {
		return err
	}

	verifyKey, err := idp.getVerificationKey()
	if err != nil {
		return err
	}

	consentKey, err := idp.getConsentKey()
	if err != nil {
		return err
	}

	clients, err := idp.hc.Client.GetClients()
	if err != nil {
		return err
	}

	idp.cache.Set(VerifyPublicKey, verifyKey, cache.DefaultExpiration)
	idp.cache.Set(ConsentPrivateKey, consentKey, cache.DefaultExpiration)
	idp.cache.Set(ClientInfo, clients, idp.config.ClientCacheExpiration)

	return err
}

// Parse and verify the challenge JWT
func (idp *IDP) getChallengeToken(challengeString string) (*jwt.Token, error) {
	token, err := jwt.Parse(challengeString, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodRSA)
		if !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		publicKey, err := idp.GetVerificationKey()
		if err != nil {
			log.Println("GetVerificationKey(): %s", err.Error())
		}
		return publicKey, err
	})

	if err != nil {
		return nil, fmt.Errorf("jwt.Parse(): %s", err.Error())
	}

	if !token.Valid {
		return nil, fmt.Errorf("Empty token")
	}

	return token, nil
}

func (idp *IDP) GetConsentKey() (*rsa.PrivateKey, error) {
	data, ok := idp.cache.Get(ConsentPrivateKey)
	if !ok {
		return nil, ErrorNotInCache
	}

	key, ok := data.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrorBadKey
	}

	return key, nil
}

func (idp *IDP) GetVerificationKey() (*rsa.PublicKey, error) {
	data, ok := idp.cache.Get(VerifyPublicKey)
	if !ok {
		return nil, ErrorNotInCache
	}

	key, ok := data.(*rsa.PublicKey)
	if !ok {
		return nil, ErrorBadKey
	}

	return key, nil
}

func (idp *IDP) GetClient(clientID string) (*hclient.Client, error) {
	data, ok := idp.cache.Get(ClientInfo)
	if !ok {
		return nil, ErrorNotInCache
	}

	clients, ok := data.(map[string]*hclient.Client)
	if !ok {
		return nil, ErrorNotInCache
	}

	client, ok := clients[clientID]
	if !ok {
		return nil, ErrorNoSuchClient
	}

	return client, nil
}

func (idp *IDP) NewChallenge(r *http.Request, user string) (challenge *Challenge, err error) {
	tokenStr := r.FormValue("challenge")
	if tokenStr == "" {
		// No challenge token
		err = ErrorBadRequest
		return
	}

	token, err := idp.getChallengeToken(tokenStr)
	if err != nil {
		// Most probably, token can't be verified or parsed
		return nil, fmt.Errorf("getChallengeToken(): %s", err.Error())
	}
	claims := token.Claims.(jwt.MapClaims)

	challenge = new(Challenge)
	challenge.Expires = time.Unix(int64(claims["exp"].(float64)), 0)
	if challenge.Expires.Before(time.Now()) {
		challenge = nil
		err = ErrorChallengeExpired
		return
	}

	// Get data from the challenge jwt
	challenge.Client, err = idp.GetClient(claims["aud"].(string))
	if err != nil {
		return nil, err
	}

	challenge.Redirect = claims["redir"].(string)

	challenge.User = user
	challenge.idp = idp

	scopes := claims["scp"].([]interface{})
	challenge.Scopes = make([]string, len(scopes), len(scopes))
	for i, scope := range scopes {
		challenge.Scopes[i] = scope.(string)
	}

	return
}

func (idp *IDP) GetChallenge(r *http.Request) (*Challenge, error) {
	session, err := idp.config.ChallengeStore.Get(r, SessionCookieName)
	if err != nil {
		return nil, err
	}

	challenge, ok := session.Values[SessionCookieName].(*Challenge)
	if !ok {
		return nil, ErrorBadChallengeCookie
	}

	if challenge.Expires.Before(time.Now()) {
		return nil, ErrorChallengeExpired
	}

	challenge.idp = idp

	return challenge, nil
}

func (idp *IDP) Close() {
	fmt.Println("IDP closed")
	idp.client = nil

	// Removes all keys from the cache
	idp.cache.Flush()
}
