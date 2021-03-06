/*
 * Copyright (c) 2020, VRAI Labs and/or its affiliates. All rights reserved.
 *
 * This software is licensed under the Apache License, Version 2.0 (the
 * "License") as published by the Apache Software Foundation.
 *
 * You may not use this file except in compliance with the License. You may
 * obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

package core

import (
	"github.com/supertokens/supertokens-go/supertokens/errors"
)

// Config used to set locations of SuperTokens instances
func Config(hosts string, apiKey string) {
	InitQuerier(hosts, apiKey)
}

// SessionInfo carrier of session token information
type SessionInfo struct {
	// some of the fields are points cause they can be nil too
	Handle         string
	UserID         string
	UserDataInJWT  map[string]interface{}
	AccessToken    *TokenInfo
	RefreshToken   *TokenInfo
	IDRefreshToken *TokenInfo
	AntiCsrfToken  *string
}

// TokenInfo carrier of cookie related info for a token
type TokenInfo struct {
	Token        string
	Expiry       uint64
	CreatedTime  uint64
	CookiePath   string
	CookieSecure bool
	Domain       *string
	SameSite     string
}

// CreateNewSession function used to create a new SuperTokens session
func CreateNewSession(userID string, jwtPayload map[string]interface{},
	sessionData map[string]interface{}) (SessionInfo, error) {
	response, err := GetQuerierInstance().SendPostRequest("newsession", "/session",
		map[string]interface{}{
			"userId":             userID,
			"userDataInJWT":      jwtPayload,
			"userDataInDatabase": sessionData,
		})
	if err != nil {
		return SessionInfo{}, err
	}
	return convertJSONResponseToSessionInfo(response), nil
}

// GetSession function used to verify a session
func GetSession(accessToken string, antiCsrfToken *string, doAntiCsrfCheck bool) (SessionInfo, error) {
	{
		handShakeInfo, handShakeError := GetHandshakeInfoInstance()
		if handShakeError != nil {
			return SessionInfo{}, handShakeError
		}
		if handShakeInfo.JwtSigningPublicKeyExpiryTime > getCurrTimeInMS() {
			accessTokenInfo, accessTokenError := getInfoFromAccessToken(accessToken,
				handShakeInfo.JwtSigningPublicKey, handShakeInfo.EnableAntiCsrf && doAntiCsrfCheck)
			if accessTokenError == nil {
				if handShakeInfo.EnableAntiCsrf && doAntiCsrfCheck &&
					(antiCsrfToken == nil || accessTokenInfo.antiCsrfToken == nil ||
						*antiCsrfToken != *(accessTokenInfo.antiCsrfToken)) {
					// we continue querying the core...
				} else {
					if !handShakeInfo.AccessTokenBlacklistingEnabled &&
						accessTokenInfo.parentRefreshTokenHash1 == nil {
						return SessionInfo{
							Handle:         accessTokenInfo.sessionHandle,
							UserID:         accessTokenInfo.userID,
							UserDataInJWT:  accessTokenInfo.userData,
							AccessToken:    nil,
							RefreshToken:   nil,
							IDRefreshToken: nil,
							AntiCsrfToken:  nil,
						}, nil
					}
					// we continue querying the core...
				}
			} else {
				if !errors.IsTryRefreshTokenError(accessTokenError) {
					return SessionInfo{}, accessTokenError
				}
				// we continue querying the core...
			}
		}
	}

	GetProcessStateInstance().AddState(CallingServiceInVerify)

	body := map[string]interface{}{
		"accessToken":     accessToken,
		"doAntiCsrfCheck": doAntiCsrfCheck,
	}
	if antiCsrfToken != nil {
		body["antiCsrfToken"] = *antiCsrfToken
	}
	response, err := GetQuerierInstance().SendPostRequest("verify", "/session/verify", body)
	if err != nil {
		return SessionInfo{}, err
	}
	if response["status"] == "OK" {
		handShakeInfo, handShakeError := GetHandshakeInfoInstance()
		if handShakeError != nil {
			if err != nil {
				return SessionInfo{}, handShakeError
			}
		}
		handShakeInfo.UpdateJwtSigningPublicKeyInfo(
			response["jwtSigningPublicKey"].(string), uint64(response["jwtSigningPublicKeyExpiryTime"].(float64)))
		return convertJSONResponseToSessionInfo(response), nil
	} else if response["status"] == "UNAUTHORISED" {
		return SessionInfo{}, errors.UnauthorizedError{
			Msg: response["message"].(string),
		}
	} else {
		return SessionInfo{}, errors.TryRefreshTokenError{
			Msg: response["message"].(string),
		}
	}
}

// RefreshSession function used to refresh a session
func RefreshSession(refreshToken string, antiCsrfToken *string) (SessionInfo, error) {
	body := map[string]interface{}{
		"refreshToken": refreshToken,
	}
	if antiCsrfToken != nil {
		body["antiCsrfToken"] = *antiCsrfToken
	}
	response, err := GetQuerierInstance().SendPostRequest("refresh", "/session/refresh", body)
	if err != nil {
		return SessionInfo{}, err
	}
	if response["status"] == "OK" {
		return convertJSONResponseToSessionInfo(response), nil
	} else if response["status"] == "UNAUTHORISED" {
		return SessionInfo{}, errors.UnauthorizedError{
			Msg: response["message"].(string),
		}
	} else {
		return SessionInfo{}, errors.TokenTheftDetectedError{
			Msg:           "Token theft detected",
			SessionHandle: (response["session"].(map[string]interface{}))["handle"].(string),
			UserID:        (response["session"].(map[string]interface{}))["userId"].(string),
		}
	}
}

// RevokeAllSessionsForUser function used to revoke all sessions for a user
func RevokeAllSessionsForUser(userID string) ([]string, error) {
	response, err := GetQuerierInstance().SendPostRequest("revokeall", "/session/remove",
		map[string]interface{}{
			"userId": userID,
		})
	if err != nil {
		return nil, err
	}
	return convertInterfaceArrayToStringArray(
		response["sessionHandlesRevoked"].([]interface{})), nil
}

// GetAllSessionHandlesForUser function used to get all sessions for a user
func GetAllSessionHandlesForUser(userID string) ([]string, error) {
	response, err := GetQuerierInstance().SendGetRequest("getall", "/session/user",
		map[string]string{
			"userId": userID,
		})
	if err != nil {
		return nil, err
	}
	return convertInterfaceArrayToStringArray(
		response["sessionHandles"].([]interface{})), nil
}

// RevokeSession function used to revoke a specific session
func RevokeSession(sessionHandle string) (bool, error) {
	response, err := GetQuerierInstance().SendPostRequest("revoke", "/session/remove",
		map[string]interface{}{
			"sessionHandles": [1]string{sessionHandle},
		})
	if err != nil {
		return false, err
	}
	return len(response["sessionHandlesRevoked"].([]interface{})) == 1, nil
}

// RevokeMultipleSessions function used to revoke a list of sessions
func RevokeMultipleSessions(sessionHandles []string) ([]string, error) {
	response, err := GetQuerierInstance().SendPostRequest("revokemultiple", "/session/remove",
		map[string]interface{}{
			"sessionHandles": sessionHandles,
		})
	if err != nil {
		return nil, err
	}
	return convertInterfaceArrayToStringArray(
		response["sessionHandlesRevoked"].([]interface{})), nil
}

// GetSessionData function used to get session data for the given handle
func GetSessionData(sessionHandle string) (map[string]interface{}, error) {
	response, err := GetQuerierInstance().SendGetRequest("getsessiondata", "/session/data",
		map[string]string{
			"sessionHandle": sessionHandle,
		})
	if err != nil {
		return nil, err
	}
	if response["status"] == "OK" {
		return response["userDataInDatabase"].(map[string]interface{}), nil
	}
	return nil, errors.UnauthorizedError{
		Msg: response["message"].(string),
	}
}

// UpdateSessionData function used to update session data for the given handle
func UpdateSessionData(sessionHandle string, newSessionData map[string]interface{}) error {
	response, err := GetQuerierInstance().SendPutRequest("updatesessiondata", "/session/data",
		map[string]interface{}{
			"sessionHandle":      sessionHandle,
			"userDataInDatabase": newSessionData,
		})
	if err != nil {
		return err
	}
	if response["status"] == "UNAUTHORISED" {
		return errors.UnauthorizedError{
			Msg: response["message"].(string),
		}
	}
	return nil
}

// GetJWTPayload function used to get jwt payload for the given handle
func GetJWTPayload(sessionHandle string) (map[string]interface{}, error) {
	response, err := GetQuerierInstance().SendGetRequest("getjwtpayload", "/jwt/data",
		map[string]string{
			"sessionHandle": sessionHandle,
		})
	if err != nil {
		return nil, err
	}
	if response["status"] == "OK" {
		return response["userDataInJWT"].(map[string]interface{}), nil
	}
	return nil, errors.UnauthorizedError{
		Msg: response["message"].(string),
	}
}

// UpdateJWTPayload function used to update jwt payload for the given handle
func UpdateJWTPayload(sessionHandle string, newJWTPayload map[string]interface{}) error {
	response, err := GetQuerierInstance().SendPutRequest("updatejwtpayload", "/jwt/data",
		map[string]interface{}{
			"sessionHandle": sessionHandle,
			"userDataInJWT": newJWTPayload,
		})
	if err != nil {
		return err
	}
	if response["status"] == "UNAUTHORISED" {
		return errors.UnauthorizedError{
			Msg: response["message"].(string),
		}
	}
	return nil
}

// RegenerateSession function used to regenerate a session
func RegenerateSession(accessToken string, newJWTPayload map[string]interface{}) (SessionInfo, error) {
	response, err := GetQuerierInstance().SendPostRequest("regenerate", "/session/regenerate",
		map[string]interface{}{
			"accessToken":   accessToken,
			"userDataInJWT": newJWTPayload,
		})
	if err != nil {
		return SessionInfo{}, err
	}
	if response["status"] == "UNAUTHORISED" {
		return SessionInfo{}, errors.UnauthorizedError{
			Msg: response["message"].(string),
		}
	}
	return convertJSONResponseToSessionInfo(response), nil
}
