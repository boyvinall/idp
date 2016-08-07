package core

import (
	"errors"
)

var (
	ErrorAuthenticationFailure = errors.New("authentication failure")
	ErrorNoSuchUser            = errors.New("no such user")
	ErrorUserAlreadyExists     = errors.New("user already exists")
	ErrorBadRequest            = errors.New("bad request")
	ErrorBadChallengeToken     = errors.New("bad challenge token format")
	ErrorNoChallengeCookie     = errors.New("challenge token isn't stored in a cookie")
	ErrorBadChallengeCookie    = errors.New("bad format of the challenge cookie")
	ErrorChallengeExpired      = errors.New("challenge expired")
	ErrorNoKey                 = errors.New("there's no key in the cache")
	ErrorNoSuchClient          = errors.New("there's no OIDC Client with such id")
	ErrorBadKey                = errors.New("bad key stored in the cache ")
	ErrorInvalidConfig         = errors.New("invalid config")
	ErrorBadPublicKey          = errors.New("cannot convert to public key")
	ErrorBadPrivateKey         = errors.New("cannot convert to private key")
	ErrorNotInCache            = errors.New("cache doesn't have the requested data")
	ErrorSessionExpired        = errors.New("session expired")
	ErrorInternalError         = errors.New("server internal error")
	ErrorPasswordMismatch      = errors.New("passwords don't match")
	ErrorComplexityFailed      = errors.New("complexity failed")
	ErrorNotImplemented        = errors.New("not implemented")
	ErrorNoSuchEntry           = errors.New("there's no entry for a gived id")
)
