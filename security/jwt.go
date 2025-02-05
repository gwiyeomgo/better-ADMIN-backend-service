package security

import (
	"better-admin-backend-service/config"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

// https://docs.apigee.com/api-platform/reference/policies/oauth-http-status-code-reference
var InvalidAccessToken = errors.New("invalid access token")
var AccessTokenExpired = errors.New("access token expired")

type JwtAuthentication struct {
}

func (JwtAuthentication) GenerateJwtToken(claim UserClaim) (JwtToken, error) {
	claimMap, err := claim.ConvertMap()
	if err != nil {
		return JwtToken{}, err
	}

	accessTokenClaims := jwt.MapClaims{}
	for key, value := range claimMap {
		accessTokenClaims[key] = value
	}

	accessTokenClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims).SignedString([]byte(config.Config.JwtSecret))

	if err != nil {
		return JwtToken{}, errors.Wrap(err, "create accessToken error")
	}

	refreshTokenClaims := jwt.MapClaims{}
	for key, value := range claimMap {
		refreshTokenClaims[key] = value
	}

	refreshTokenExpires := time.Now().Add(time.Hour * 24 * 7)
	refreshTokenClaims["exp"] = refreshTokenExpires.Unix()
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims).SignedString([]byte(config.Config.JwtSecret))

	if err != nil {
		return JwtToken{}, errors.Wrap(err, "create refreshToken error")
	}

	return JwtToken{
		AccessToken:         accessToken,
		RefreshToken:        refreshToken,
		RefreshTokenExpires: refreshTokenExpires,
	}, nil
}

func (JwtAuthentication) GenerateJwtAccessTokenNeverExpired(claim UserClaim) (string, error) {
	claimMap, err := claim.ConvertMap()
	if err != nil {
		return "", err
	}

	accessTokenClaims := jwt.MapClaims{}
	for key, value := range claimMap {
		accessTokenClaims[key] = value
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims).SignedString([]byte(config.Config.JwtSecret))

	if err != nil {
		return "", errors.Wrap(err, "create accessToken error")
	}

	return accessToken, nil
}

func (JwtAuthentication) ConvertTokenUserClaim(token string) (*UserClaim, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) { return []byte(config.Config.JwtSecret), nil })

	if err != nil {
		log.Error("JWT parsing error: " + err.Error())
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				return nil, AccessTokenExpired
			}
		}
		return nil, InvalidAccessToken
	}

	if jwt.SigningMethodHS256.Alg() != parsedToken.Header["alg"] {
		log.Error(fmt.Sprintf("Error: jwt token is expected %s signing method but token specified %s",
			jwt.SigningMethodHS256.Alg(), parsedToken.Header["alg"]))
		return nil, InvalidAccessToken
	}

	if !parsedToken.Valid {
		return nil, InvalidAccessToken
	}

	claimInfo, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		log.Error("Can'get jwt.MapClaims")
		return nil, InvalidAccessToken
	}

	userClaim, err := NewUserClaim(claimInfo)
	if err != nil {
		return nil, err
	}

	return &userClaim, nil
}

func (jwtAuthentication JwtAuthentication) RefreshAccessToken(refreshToken string) (string, error) {
	userClaim, err := jwtAuthentication.ConvertTokenUserClaim(refreshToken)
	if err != nil {
		return "", err
	}

	jwtToken, err := jwtAuthentication.GenerateJwtToken(*userClaim)
	if err != nil {
		return "", err
	}

	return jwtToken.AccessToken, nil
}

func (jwtAuthentication JwtAuthentication) ValidateToken(token string) error {
	_, err := jwtAuthentication.ConvertTokenUserClaim(token)
	return err
}

type JwtToken struct {
	AccessToken         string
	RefreshToken        string
	RefreshTokenExpires time.Time
}

func (t JwtToken) GetRefreshTokenExpiresForCookie() time.Time {
	// 쿠키의 Expire 시간을 Local time 설정하면 브라우저 쿠키의 Expire 에는 UTC 기준으로 설정됨(KST인 경우 -9 시간)
	// 이를 막기 위해 timezone 의 offset 을 구하여 현재 Local time 에 offset 시간을 더해줌.
	_, offset := t.RefreshTokenExpires.Zone()
	return t.RefreshTokenExpires.Add(time.Duration(offset) * time.Second)
}

type UserClaim struct {
	Id          uint     `json:"id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func (c UserClaim) ConvertMap() (map[string]interface{}, error) {
	bytes, err := json.Marshal(c)

	if err != nil {
		return nil, errors.Wrap(err, "JSON Marshal error")
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(bytes, &resultMap); err != nil {
		return nil, errors.Wrap(err, "JSON Unmarshal error")
	}

	return resultMap, nil
}

func NewUserClaim(mapUserClaim map[string]interface{}) (UserClaim, error) {
	bytes, err := json.Marshal(mapUserClaim)
	if err != nil {
		return UserClaim{}, errors.Wrap(err, "JSON Marshal error")
	}

	var claim UserClaim
	if err := json.Unmarshal(bytes, &claim); err != nil {
		return UserClaim{}, errors.Wrap(err, "JSON Unmarshal error")
	}

	return claim, nil
}
