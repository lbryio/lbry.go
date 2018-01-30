package query

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/null.go"
)

func InterpolateParams(query string, args ...interface{}) (string, error) {
	for i := 0; i < len(args); i++ {
		field := reflect.ValueOf(args[i])

		if value, ok := field.Interface().(time.Time); ok {
			query = strings.Replace(query, "?", `"`+value.Format("2006-01-02 15:04:05")+`"`, 1)
		} else if nullable, ok := field.Interface().(null.Nullable); ok {
			if nullable.IsNull() {
				query = strings.Replace(query, "?", "NULL", 1)
			} else {
				switch field.Type() {
				case reflect.TypeOf(null.Time{}):
					query = strings.Replace(query, "?", `"`+field.Interface().(null.Time).Time.Format("2006-01-02 15:04:05")+`"`, 1)
				case reflect.TypeOf(null.Int{}):
					query = strings.Replace(query, "?", strconv.FormatInt(int64(field.Interface().(null.Int).Int), 10), 1)
				case reflect.TypeOf(null.Int8{}):
					query = strings.Replace(query, "?", strconv.FormatInt(int64(field.Interface().(null.Int8).Int8), 10), 1)
				case reflect.TypeOf(null.Int16{}):
					query = strings.Replace(query, "?", strconv.FormatInt(int64(field.Interface().(null.Int16).Int16), 10), 1)
				case reflect.TypeOf(null.Int32{}):
					query = strings.Replace(query, "?", strconv.FormatInt(int64(field.Interface().(null.Int32).Int32), 10), 1)
				case reflect.TypeOf(null.Int64{}):
					query = strings.Replace(query, "?", strconv.FormatInt(field.Interface().(null.Int64).Int64, 10), 1)
				case reflect.TypeOf(null.Uint{}):
					query = strings.Replace(query, "?", strconv.FormatUint(uint64(field.Interface().(null.Uint).Uint), 10), 1)
				case reflect.TypeOf(null.Uint8{}):
					query = strings.Replace(query, "?", strconv.FormatUint(uint64(field.Interface().(null.Uint8).Uint8), 10), 1)
				case reflect.TypeOf(null.Uint16{}):
					query = strings.Replace(query, "?", strconv.FormatUint(uint64(field.Interface().(null.Uint16).Uint16), 10), 1)
				case reflect.TypeOf(null.Uint32{}):
					query = strings.Replace(query, "?", strconv.FormatUint(uint64(field.Interface().(null.Uint32).Uint32), 10), 1)
				case reflect.TypeOf(null.Uint64{}):
					query = strings.Replace(query, "?", strconv.FormatUint(field.Interface().(null.Uint64).Uint64, 10), 1)
				case reflect.TypeOf(null.String{}):
					query = strings.Replace(query, "?", `"`+field.Interface().(null.String).String+`"`, 1)
				case reflect.TypeOf(null.Bool{}):
					if field.Interface().(null.Bool).Bool {
						query = strings.Replace(query, "?", "1", 1)
					} else {
						query = strings.Replace(query, "?", "0", 1)
					}
				}
			}
		} else {
			switch field.Kind() {
			case reflect.Bool:
				boolString := "0"
				if field.Bool() {
					boolString = "1"
				}
				query = strings.Replace(query, "?", boolString, 1)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				query = strings.Replace(query, "?", strconv.FormatInt(field.Int(), 10), 1)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				query = strings.Replace(query, "?", strconv.FormatUint(field.Uint(), 10), 1)
			case reflect.Float32, reflect.Float64:
				query = strings.Replace(query, "?", strconv.FormatFloat(field.Float(), 'f', -1, 64), 1)
			case reflect.String:
				query = strings.Replace(query, "?", `"`+field.String()+`"`, 1)
			default:
				return "", errors.Err("dont know how to interpolate type " + field.Type().String())
			}
		}
	}

	// tabs to spaces, for easier copying into mysql prompt
	query = strings.Replace(query, "\t", "    ", -1)

	return query, nil
}
