package ep

import "github.com/golang-jwt/jwt"
import "github.com/gin-gonic/gin"

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

//GetUidByContext 获取请求中的uid
func GetUidByContext(c *gin.Context, TokenSecret string) string {
	token, _ := c.Cookie("token")
	if token == "" {
		return ""
	}

	//获取uid
	uid := GetDetailByToken("uid", token, TokenSecret)
	if uid == "" {
		return ""
	}

	return uid
}
