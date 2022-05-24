package ep

import "github.com/golang-jwt/jwt"

//jwt 的token解析获取用户信息
//detailName ：uid，lang
func GetDetailByToken(detailName, token, tokenSecret string) string {
	if detailName == "" {
		return ""
	}

	_token, _ := jwt.Parse(token, func(_token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})

	if _token == nil || !_token.Valid {
		return ""
	}

	claims, _ := _token.Claims.(jwt.MapClaims)

	return claims[detailName].(string)
}
