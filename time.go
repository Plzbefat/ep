package main

import (
	"database/sql/driver"
	"fmt"
	"log"
	"strings"
	"time"
)

type JsonDate struct {
	Time time.Time
}

func (t JsonDate) Str() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.Format("2006-01-02")
}

func (t JsonDate) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte(`""`), nil
	}
	formatted := fmt.Sprintf("\"%s\"", t.Time.Format("2006-01-02"))
	return []byte(formatted), nil
}

func (t *JsonDate) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		*t = JsonDate{time.Time{}}
	} else {
		if now, err := time.ParseInLocation("2006-01-02", string(data), time.Local); err == nil {
			*t = JsonDate{now}
			return nil
		}

		if now, err := time.ParseInLocation("2006-01-02", strings.Trim(string(data), `"`), time.Local); err == nil {
			*t = JsonDate{now}
			return nil
		}

	}
	return nil
}

func (t *JsonDate) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = JsonDate{Time: value}
		return nil
	}

	tTime, err := time.ParseInLocation("2006-01-02", v.(string), time.Local)
	if err == nil {
		*t = JsonDate{Time: tTime}
		return nil
	}

	return fmt.Errorf("can not convert %v to timestamp", v)
}

func (t JsonDate) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

type JsonDateTime struct {
	time.Time
}

func (t JsonDateTime) Str() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

func (t JsonDateTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	formatted := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

func (t *JsonDateTime) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		*t = JsonDateTime{time.Time{}}
	} else {
		if now, err := time.ParseInLocation("2006-01-02 15:04:05", string(data), time.Local); err == nil {
			*t = JsonDateTime{now}
			return nil
		}

		if now, err := time.ParseInLocation("2006-01-02 15:04:05", strings.Trim(string(data), `"`), time.Local); err == nil {
			*t = JsonDateTime{now}
			return nil
		}
	}
	return nil
}

func (t *JsonDateTime) Scan(v interface{}) error {
	timeStr := ""
	switch s := v.(type) {
	case time.Time:
		*t = JsonDateTime{Time: s}
		return nil
	case string:
		timeStr = s
	case []uint8:
		timeStr = string(s)
	default:
		log.Println("time scan err")
		return nil
	}

	tTime, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local)
	if err == nil {
		*t = JsonDateTime{Time: tTime}
		return nil
	}

	return fmt.Errorf("can not convert %v to timestamp", v)
}

func (t JsonDateTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

type JsonTime struct {
	time.Time
}

func (t JsonTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	formatted := fmt.Sprintf("\"%s\"", t.Format("15:04:05"))
	return []byte(formatted), nil
}

func (t *JsonTime) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		*t = JsonTime{time.Time{}}
	} else {
		if now, err := time.ParseInLocation("15:04:05", string(data), time.Local); err == nil {
			*t = JsonTime{now}
			return nil
		}

		if now, err := time.ParseInLocation("15:04:05", strings.Trim(string(data), `"`), time.Local); err == nil {
			*t = JsonTime{now}
			return nil
		}
	}
	return nil
}

func (t *JsonTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = JsonTime{Time: value}
		return nil
	}

	tt := v.([]uint8)

	tTime, err := time.ParseInLocation("15:04:05", string(tt), time.Local)
	if err == nil {
		*t = JsonTime{Time: tTime}
		return nil
	}

	return fmt.Errorf("can not convert %v to timestamp", v)
}

func (t JsonTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}
