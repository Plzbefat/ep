//gin反馈包装
package ep

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HTTP status codes as registered with IANA.
// See: https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
func R(c *gin.Context, status int, success bool, msg string, obj ...interface{}) {
	if len(obj) > 0 {
		c.JSON(status, gin.H{"success": success, "msg": msg, "result": obj[0]})
	} else {
		c.JSON(status, gin.H{"success": success, "msg": msg})
	}
}

func RF(c *gin.Context, msg ...interface{}) {
	if len(msg) > 0 {
		switch msg[0].(type) {
		case error:
			R(c, http.StatusOK, false, msg[0].(error).Error())
		case string:
			R(c, http.StatusOK, false, msg[0].(string))
		}
	} else {
		R(c, http.StatusOK, false, "")
	}
}

func RT(c *gin.Context, msg string, data ...interface{}) {
	if len(data) > 0 {
		switch data[0].(type) {
		case gin.H:
			R(c, http.StatusOK, true, msg, data[0].(gin.H))
		default:
			R(c, http.StatusOK, true, msg, gin.H{"data": data[0]})
		}
	} else {
		R(c, http.StatusOK, true, msg)
	}
}
