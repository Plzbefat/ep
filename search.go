package ep

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type search struct {
	// db表
	db *gorm.DB
	// context
	c *gin.Context
	//搜索用表
	model interface{}
	//表名
	table string
	//反馈数据
	Out interface{}
	//总行数
	total int64

	error error

	// sql 限制
	sqlLimit     string
	sqlLimitArgs []interface{}

	searchParams struct {
		Key      string `json:"key" form:"key"` //开启模糊搜索 只要有这个字段就行
		Precise  bool   `json:"precise" form:"precise"`
		Current  int    `json:"current" form:"current"`
		PageSize int    `json:"pageSize" form:"pageSize"`
		Order    string `json:"order" form:"order"`
		Start    string `json:"start" form:"start"`
		Stop     string `json:"stop" form:"stop"`
		Offset   int    `json:"-" form:"-"`
	}
}

//Search 新的分页搜索
func Search(c *gin.Context) *search {
	return &search{c: c}
}

func (s *search) DB(db *gorm.DB) *search {
	s.db = db
	return s
}

func (s *search) Table(tableName string) *search {
	s.table = tableName
	return s
}
func (s *search) Model(model interface{}) *search {
	s.model = model
	return s
}

func (s *search) Limit(limit string, sqlLimitArgs ...interface{}) *search {
	s.sqlLimit = limit
	if len(sqlLimitArgs) > 0 {
		s.sqlLimitArgs = sqlLimitArgs
	}
	return s
}

//获取 分页排序日期 基础字段信息
func (s *search) getSearchParam() *search {
	if s.db == nil {
		s.error = errors.New("database is empty")
		return s
	}

	if s.error = s.c.ShouldBindQuery(&s.searchParams); s.error != nil {
		return s
	}

	if s.searchParams.Current == 0 {
		s.searchParams.Current = 1
	}

	if s.searchParams.PageSize == 0 {
		s.searchParams.PageSize = 10
	}

	if off := s.searchParams.Current - 1; off > 0 {
		s.searchParams.Offset = s.searchParams.PageSize * off
	}

	s.error = s.c.ShouldBindQuery(s.model)
	return s
}

func (s *search) getData() *search {
	if s.error != nil {
		return s
	}

	var query *gorm.DB
	if s.table != "" {
		query = s.db.Table(s.table)
	} else {
		query = s.db.Model(s.model)
	}

	sqlWhere := ""

	//全局模糊搜索
	if s.searchParams.Key != "" {
		//模糊搜索字段
		searchModelRef := reflect.ValueOf(s.model).Elem()
		for i := 0; i < searchModelRef.NumField(); i++ {
			var value string

			fieldType := searchModelRef.Field(i).Type().String()

			//只能是字符串类的做模糊搜索
			switch fieldType {
			case "string":
				value = s.searchParams.Key
			default:
				continue
			}

			if value == "" {
				continue
			}

			key := searchModelRef.Type().Field(i).Tag.Get("form")
			if key == "" {
				continue
			}

			if sqlWhere != "" {
				sqlWhere += " or "
			}

			//数值类的精准搜索
			if fieldType == "int" || fieldType == "int64" || fieldType == "float64" {
				sqlWhere += fmt.Sprintf(" %s = %s", key, value)
			} else {
				//字符串 手动控制 精准搜索
				if s.searchParams.Precise {
					sqlWhere += fmt.Sprintf(" %s = '%s' ", key, value)
				} else {
					sqlWhere += fmt.Sprintf(" %s like '%%%s%%' ", key, value)
				}
			}
		}
	} else {
		//模糊搜索字段
		searchModelRef := reflect.ValueOf(s.model).Elem()
		for i := 0; i < searchModelRef.NumField(); i++ {
			var value string

			fieldType := searchModelRef.Field(i).Type().String()

			switch fieldType {
			case "db.JsonDate":
				value = searchModelRef.Field(i).Interface().(JsonDate).Str()
			case "db.JsonDateTime":
				value = searchModelRef.Field(i).Interface().(JsonDateTime).Str()
			case "int":
				value = strconv.Itoa(searchModelRef.Field(i).Interface().(int))
				if value == "0" {
					value = ""
				}
			case "int64":
				value = strconv.FormatInt(searchModelRef.Field(i).Interface().(int64), 10)
				if value == "0" {
					value = ""
				}
			case "float64":
				value = strconv.Itoa(int(searchModelRef.Field(i).Float()))
			case "string":
				value = searchModelRef.Field(i).String()
			}

			if value == "" {
				continue
			}

			key := searchModelRef.Type().Field(i).Tag.Get("form")
			if key == "" {
				continue
			}

			if sqlWhere != "" {
				sqlWhere += " and "
			}

			//数值类的精准搜索
			if fieldType == "int" || fieldType == "int64" || fieldType == "float64" || fieldType == "bool" {
				sqlWhere += fmt.Sprintf(" %s = %s", key, value)
			} else {
				//字符串 手动控制 精准搜索
				if s.searchParams.Precise {
					sqlWhere += fmt.Sprintf(" %s = '%s' ", key, value)
				} else {
					sqlWhere += fmt.Sprintf(" %s like '%%%s%%' ", key, value)
				}
			}
		}

		//精准搜索 但是 sql 为空 那就不反馈数据
		if s.searchParams.Precise && sqlWhere == "" {
			return s
		}
	}

	query = query.Where(sqlWhere)

	//limit
	if s.sqlLimit != "" {
		if s.sqlLimitArgs != nil {
			query = query.Where(s.sqlLimit, s.sqlLimitArgs...)
		} else {
			query = query.Where(s.sqlLimit)
		}
	}

	//date between
	if s.searchParams.Start != "" && s.searchParams.Stop != "" {
		query = query.Where(fmt.Sprintf(" created_at between '%s 00:00:00' and '%s 23:59:59' ", s.searchParams.Start, s.searchParams.Stop))
	}

	//统计查询总量
	query.Count(&s.total)

	//分页排序参数
	offset, limit, order := s.searchParams.Offset, s.searchParams.PageSize, s.searchParams.Order
	s.error = query.Offset(offset).Limit(limit).Order(order).Scan(&s.Out).Error
	return s
}

func (s *search) Scan(out interface{}) *search {
	s.Out = out
	return s.getSearchParam().getData()
}

//Resp 反馈信息
func (s *search) Resp() {
	if s.error != nil {
		RF(s.c, s.error.Error())
	} else {
		RT(s.c, "", gin.H{"data": s.Out, "total": s.total})
	}
}
