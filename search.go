package ep

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SearchStruct struct {
	// db表
	db *gorm.DB
	// context
	c *gin.Context
	//搜索用表
	model interface{}
	//表名
	table string
	//反馈数据
	Data interface{} `json:"data"`
	//总行数
	Total int64 `json:"total"`
	//精准搜索
	precise bool
	//是否不统计项目
	notCountTotal bool
	//
	_select string

	error error

	// sql 限制
	sqlWhere     string
	sqlLimitArgs []interface{}

	searchParams struct {
		Key      string `json:"key" form:"key"`         //开启模糊搜索 只要有这个字段就行
		Precise  bool   `json:"precise" form:"precise"` //是否为精准搜索
		Current  int    `json:"current" form:"current"`
		PageSize int    `json:"pageSize" form:"pageSize"`
		Order    string `json:"order" form:"order"`
		Between  string `json:"between" form:"between"`
		Offset   int    `json:"-" form:"-"`
	}
}

// Search 新的分页搜索
func Search(c *gin.Context) *SearchStruct {
	return &SearchStruct{c: c}
}

func (s *SearchStruct) DB(db *gorm.DB) *SearchStruct {
	s.db = db
	return s
}

func (s *SearchStruct) Table(tableName string) *SearchStruct {
	s.table = tableName
	return s
}

func (s *SearchStruct) Select(_select string) *SearchStruct {
	s._select = _select
	return s
}

// 精准搜索
func (s *SearchStruct) Precise() *SearchStruct {
	s.precise = true
	return s
}

// 不统计总数
func (s *SearchStruct) NoCountTotal() *SearchStruct {
	s.notCountTotal = true
	return s
}

func (s *SearchStruct) Model(model interface{}) *SearchStruct {
	s.model = model
	return s
}

func (s *SearchStruct) Where(where string, whereArgs ...interface{}) *SearchStruct {
	s.sqlWhere = where
	if len(whereArgs) > 0 {
		s.sqlLimitArgs = whereArgs
	}
	return s
}

// 获取 分页排序日期 基础字段信息
func (s *SearchStruct) getSearchParam() *SearchStruct {
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

func (s *SearchStruct) getData() *SearchStruct {
	if s.error != nil {
		return s
	}

	var query *gorm.DB

	if s._select == "" {
		s._select = "*"
	}

	if s.table != "" {
		query = s.db.Table(s.table).Select(s._select)
	} else {
		query = s.db.Model(s.model).Select(s._select)
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
				//字符串 主动/手动控制 精准搜索
				if s.precise || s.searchParams.Precise {
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
				if s.precise || s.searchParams.Precise {
					sqlWhere += fmt.Sprintf(" %s = '%s' ", key, value)
				} else {
					sqlWhere += fmt.Sprintf(" %s like '%%%s%%' ", key, value)
				}
			}
		}

		//精准搜索 但是 sql 为空 那就不反馈数据
		if (s.precise || s.searchParams.Precise) && sqlWhere == "" {
			return s
		}
	}

	query = query.Where(sqlWhere)

	//limit
	if s.sqlWhere != "" {
		if s.sqlLimitArgs != nil {
			query = query.Where(s.sqlWhere, s.sqlLimitArgs...)
		} else {
			query = query.Where(s.sqlWhere)
		}
	}

	//between
	if s.searchParams.Between != "" {
		query = query.Where(fmt.Sprintf(" %s ", s.searchParams.Between))
	}

	//统计查询总量
	if !s.notCountTotal {
		query.Count(&s.Total)
	}

	//分页排序参数
	offset, limit, order := s.searchParams.Offset, s.searchParams.PageSize, s.searchParams.Order
	s.error = query.Offset(offset).Limit(limit).Order(order).Scan(&s.Data).Error
	return s
}

func (s *SearchStruct) Scan(out interface{}) *SearchStruct {
	s.Data = out
	return s.getSearchParam().getData()
}

// Resp 反馈信息
func (s *SearchStruct) Resp() {
	if s.error != nil {
		RF(s.c, s.error.Error())
	} else {
		if !s.notCountTotal {
			RT(s.c, "", gin.H{"data": s.Data, "total": s.Total})
		} else {
			RT(s.c, "", gin.H{"data": s.Data})
		}
	}
}
