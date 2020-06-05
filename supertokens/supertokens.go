package supertokens

import (
	"net/http"
	"reflect"

	"github.com/supertokens/supertokens-go/supertokens/core"
	"github.com/supertokens/supertokens-go/supertokens/errors"
)

// Config used to set locations of SuperTokens instances
func Config(hosts string) error {
	return core.Config(hosts)
}

// CreateNewSession function used to create a new SuperTokens session
func CreateNewSession(response *http.ResponseWriter,
	userID string, payload ...map[string]interface{}) (Session, error) {
	// TODO:

	var session core.SessionInfo
	var err error
	if len(payload) == 0 {
		session, err = core.CreateNewSession(userID, map[string]interface{}{}, map[string]interface{}{})
	} else if len(payload) == 1 {
		session, err = core.CreateNewSession(userID, payload[0], map[string]interface{}{})
	} else {
		session, err = core.CreateNewSession(userID, payload[0], payload[1])
	}

	if err != nil {
		return Session{}, err
	}

	//attach token to cookies
	accessToken := session.AccessToken
	refreshToken := session.RefreshToken
	idRefreshToken := session.IDRefreshToken

	attachAccessTokenToCookie(
		response,
		accessToken.Token,
		accessToken.Expiry,
		accessToken.Domain,
		accessToken.CookiePath,
		accessToken.CookieSecure,
		accessToken.SameSite,
	)

	attachRefreshTokenToCookie(
		response,
		refreshToken.Token,
		refreshToken.Expiry,
		refreshToken.Domain,
		refreshToken.CookiePath,
		refreshToken.CookieSecure,
		refreshToken.SameSite,
	)

	setIDRefreshTokenInHeaderAndCookie(
		response,
		idRefreshToken.Token,
		idRefreshToken.Expiry,
		idRefreshToken.Domain,
		idRefreshToken.CookiePath,
		idRefreshToken.CookieSecure,
		idRefreshToken.SameSite,
	)

	if session.AntiCsrfToken != nil {
		setAntiCsrfTokenInHeaders(response, *session.AntiCsrfToken)
	}

	return Session{
		accessToken:   accessToken.Token,
		sessionHandle: session.Handle,
		userID:        session.UserID,
		userDataInJWT: session.UserDataInJWT,
		response:      response,
	}, nil

}

// GetSession function used to verify a session
func GetSession(response *http.ResponseWriter, request *http.Request,
	doAntiCsrfCheck bool) (Session, error) {
	// TODO:

	saveFrontendInfoFromRequest(request)

	accessToken := getAccessTokenFromCookie(request)

	if accessToken == nil {
		// maybe the access token has expired.
		return Session{}, errors.TryRefreshTokenError{
			Msg: "access token missing in cookies",
		}
	}

	antiCsrfToken := getAntiCsrfTokenFromHeaders(request)
	idRefreshToken := getIDRefreshTokenFromCookie(request)

	session, err := core.GetSession(*accessToken, antiCsrfToken, doAntiCsrfCheck, idRefreshToken)
	if err != nil {
		if reflect.TypeOf(err) == reflect.TypeOf(errors.UnauthorisedError{}) {
			handShakeInfo, err := core.GetHandshakeInfoInstance()
			if err != nil {
				return Session{}, err
			}
			clearSessionFromCookie(response,
				handShakeInfo.CookieDomain,
				handShakeInfo.CookieSecure,
				handShakeInfo.AccessTokenPath,
				handShakeInfo.RefreshTokenPath,
				handShakeInfo.IDRefreshTokenPath,
				handShakeInfo.CookieSameSite,
			)
		}
	}

	if session.AccessToken != nil {
		attachAccessTokenToCookie(
			response,
			session.AccessToken.Token,
			session.AccessToken.Expiry,
			session.AccessToken.Domain,
			session.AccessToken.CookiePath,
			session.AccessToken.CookieSecure,
			session.AccessToken.SameSite,
		)
	}

	return Session{
		accessToken:   *accessToken,
		response:      response,
		sessionHandle: session.Handle,
		userDataInJWT: session.UserDataInJWT,
		userID:        session.UserID,
	}, nil
}

// RefreshSession function used to refresh a session
func RefreshSession(response *http.ResponseWriter, request *http.Request) (Session, error) {
	// TODO:
	saveFrontendInfoFromRequest(request)

	inputRefreshToken := getRefreshTokenFromCookie(request)

	if inputRefreshToken == nil {

		handShakeInfo, err := core.GetHandshakeInfoInstance()
		if err != nil {
			return Session{}, err
		}

		clearSessionFromCookie(
			response,
			handShakeInfo.CookieDomain,
			handShakeInfo.CookieSecure,
			handShakeInfo.AccessTokenPath,
			handShakeInfo.RefreshTokenPath,
			handShakeInfo.IDRefreshTokenPath,
			handShakeInfo.CookieSameSite)
		return Session{}, errors.UnauthorisedError{
			Msg: "Missing auth tokens in cookies. Have you set the correct refresh API path in your frontend and SuperTokens config?",
		}
	}

	session, err := core.RefreshSession(*inputRefreshToken)

	if err != nil {

		if (reflect.TypeOf(err) == reflect.TypeOf(errors.UnauthorisedError{}) ||
			reflect.TypeOf(err) == reflect.TypeOf(errors.TokenTheftDetectedError{})) {
			handShakeInfo, err2 := core.GetHandshakeInfoInstance()
			if err2 != nil {
				return Session{}, err2
			}

			clearSessionFromCookie(
				response,
				handShakeInfo.CookieDomain,
				handShakeInfo.CookieSecure,
				handShakeInfo.AccessTokenPath,
				handShakeInfo.RefreshTokenPath,
				handShakeInfo.IDRefreshTokenPath,
				handShakeInfo.CookieSameSite)
		}
		return Session{}, err
	}

	//attach cookies
	accessToken := session.AccessToken
	refreshToken := session.RefreshToken
	idRefreshToken := session.IDRefreshToken

	attachAccessTokenToCookie(
		response,
		accessToken.Token,
		accessToken.Expiry,
		accessToken.Domain,
		accessToken.CookiePath,
		accessToken.CookieSecure,
		accessToken.SameSite,
	)

	attachRefreshTokenToCookie(
		response,
		refreshToken.Token,
		refreshToken.Expiry,
		refreshToken.Domain,
		refreshToken.CookiePath,
		refreshToken.CookieSecure,
		refreshToken.SameSite,
	)

	setIDRefreshTokenInHeaderAndCookie(
		response,
		idRefreshToken.Token,
		idRefreshToken.Expiry,
		idRefreshToken.Domain,
		idRefreshToken.CookiePath,
		idRefreshToken.CookieSecure,
		idRefreshToken.SameSite,
	)

	if session.AntiCsrfToken != nil {
		setAntiCsrfTokenInHeaders(response, *session.AntiCsrfToken)
	}

	return Session{
		accessToken:   accessToken.Token,
		sessionHandle: session.Handle,
		userID:        session.UserID,
		userDataInJWT: session.UserDataInJWT,
		response:      response,
	}, nil
}

// RevokeAllSessionsForUser function used to revoke all sessions for a user
func RevokeAllSessionsForUser(userID string) ([]string, error) {
	return core.RevokeAllSessionsForUser(userID)
}

// GetAllSessionHandlesForUser function used to get all sessions for a user
func GetAllSessionHandlesForUser(userID string) ([]string, error) {
	return core.GetAllSessionHandlesForUser(userID)
}

// RevokeSession function used to revoke a specific session
func RevokeSession(sessionHandle string) (bool, error) {
	return core.RevokeSession(sessionHandle)
}

// RevokeMultipleSessions function used to revoke a list of sessions
func RevokeMultipleSessions(sessionHandles []string) ([]string, error) {
	return core.RevokeMultipleSessions(sessionHandles)
}

// GetSessionData function used to get session data for the given handle
func GetSessionData(sessionHandle string) (map[string]interface{}, error) {
	return core.GetSessionData(sessionHandle)
}

// UpdateSessionData function used to update session data for the given handle
func UpdateSessionData(sessionHandle string, newSessionData map[string]interface{}) error {
	return core.UpdateSessionData(sessionHandle, newSessionData)
}

// SetRelevantHeadersForOptionsAPI function is used to set headers specific to SuperTokens for OPTIONS API
func SetRelevantHeadersForOptionsAPI(response *http.ResponseWriter) error {
	return core.SetRelevantHeadersForOptionsAPI(response)
}

// GetJWTPayload function used to get jwt payload for the given handle
func GetJWTPayload(sessionHandle string) (map[string]interface{}, error) {
	return core.GetJWTPayload(sessionHandle)
}

// UpdateJWTPayload function used to update jwt payload for the given handle
func UpdateJWTPayload(sessionHandle string, newJWTPayload map[string]interface{}) error {
	return core.UpdateJWTPayload(sessionHandle, newJWTPayload)
}

// OnTokenTheftDetected function to override default behaviour of handling token thefts
func OnTokenTheftDetected(handler func(string, string, http.ResponseWriter)) {
	core.GetErrorHandlersInstance().OnTokenTheftDetectedErrorHandler = handler
}

// OnUnauthorised function to override default behaviour of handling unauthorised error
func OnUnauthorised(handler func(error, http.ResponseWriter)) {
	core.GetErrorHandlersInstance().OnUnauthorisedErrorHandler = handler
}

// OnTryRefreshToken function to override default behaviour of handling try refresh token errors
func OnTryRefreshToken(handler func(error, http.ResponseWriter)) {
	core.GetErrorHandlersInstance().OnTryRefreshTokenErrorHandler = handler
}

// OnGeneralError function to override default behaviour of handling general errors
func OnGeneralError(handler func(error, http.ResponseWriter)) {
	core.GetErrorHandlersInstance().OnGeneralErrorHandler = handler
}
