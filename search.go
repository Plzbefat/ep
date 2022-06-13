package ep

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Search struct {
	//*db表
	Database *gorm.DB
	//*context
	C *gin.Context
	//*输出数据
	Data interface{}
	//*结构体地址 url 传入参数 必须带上 form 和 json 的tag!!!
	ForSearch interface{}
	Limit     string

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

	Total int64
	Error error
}

//获取 分页排序日期 基础字段信息
func (s *Search) getSearchParam() *Search {
	if s.Database == nil {
		s.Error = errors.New("database is empty")
		return s
	}

	if s.Error = s.C.ShouldBindQuery(&s.searchParams); s.Error != nil {
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

	s.Error = s.C.ShouldBindQuery(s.ForSearch)
	return s
}

//获取数据
func (s *Search) getData() *Search {
	if s.Error != nil {
		return s
	}

	query := s.Database.Model(s.Data)

	sqlWhere := ""

	//全局模糊搜索
	if s.searchParams.Key != "" {
		//模糊搜索字段
		searchModelRef := reflect.ValueOf(s.ForSearch).Elem()
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
		searchModelRef := reflect.ValueOf(s.ForSearch).Elem()
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
			case "bool":
				if value != "" {
					if searchModelRef.Field(i).Bool() {
						value = "1"
					} else {
						value = "0"
					}
				}
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
	query = query.Where(s.Limit)

	//date between
	if s.searchParams.Start != "" && s.searchParams.Stop != "" {
		query = query.Where(fmt.Sprintf(" created_at between '%s 00:00:00' and '%s 23:59:59' ", s.searchParams.Start, s.searchParams.Stop))
	}

	//统计查询总量
	query.Count(&s.Total)

	//分页排序参数
	offset, limit, order := s.searchParams.Offset, s.searchParams.PageSize, s.searchParams.Order
	s.Error = query.Offset(offset).Limit(limit).Order(order).Scan(&s.Data).Error
	return s
}

func (s *Search) Exec() *Search {
	return s.getSearchParam().getData()
}
