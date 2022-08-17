package ep

import (
	"github.com/gin-gonic/gin"
)

//gin 查询值不能为空
func QueryMustNotNull(c *gin.Context, key string) string {
	value := c.Query(key)
	if value == "" {
		RF(c, key+" 不能为空")
		return ""
	}
	return value
}

//gin 查询值在指定值内
func QueryMustInArray(c *gin.Context, key string, array []string) string {
	value := c.Query(key)

	for _, v := range array {
		if value == v {
			return v
		}
	}

	RF(c, key+" 值非法")
	return ""
}
