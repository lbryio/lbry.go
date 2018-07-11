package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/util"
	"github.com/lbryio/lbry.go/validator"
	v "github.com/lbryio/ozzo-validation"

	"github.com/spf13/cast"
)

// ResponseHeaders are returned with each response
var ResponseHeaders map[string]string

// LogError Allows specific error logging for the server at specific points.
var LogError = func(*http.Request, *Response, error) {}

// LogInfo Allows for specific logging information.
var LogInfo = func(*http.Request, *Response) {}

// TraceEnabled Attaches a trace field to the JSON response when enabled.
var TraceEnabled = false

var ErrAuthenticationRequired = errors.Base("authentication required")
var ErrNotAuthenticated = errors.Base("could not authenticate user")
var ErrForbidden = errors.Base("you are not authorized to perform this action")

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Status int
	Err    error
}

// Allows StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Response is returned by API handlers
type Response struct {
	Status      int
	Data        interface{}
	RedirectURL string
	Error       error
}

// Handler handles API requests
type Handler func(r *http.Request) Response

func (h Handler) callHandlerSafely(r *http.Request) (rsp Response) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Err("%v", r)
			}
			rsp = Response{Error: errors.Wrap(err, 2)}
		}
	}()

	return h(r)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set header settings
	if ResponseHeaders != nil {
		//Multiple readers, no writers is okay
		for key, value := range ResponseHeaders {
			w.Header().Set(key, value)
		}
	}

	// Stop here if its a preflighted OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}

	rsp := h.callHandlerSafely(r)

	if rsp.Status == 0 {
		if rsp.Error != nil {
			if statusError, ok := rsp.Error.(StatusError); ok {
				rsp.Status = statusError.Status
			} else if errors.Is(rsp.Error, ErrAuthenticationRequired) {
				rsp.Status = http.StatusUnauthorized
			} else if errors.Is(rsp.Error, ErrNotAuthenticated) || errors.Is(rsp.Error, ErrForbidden) {
				rsp.Status = http.StatusForbidden
			} else {
				rsp.Status = http.StatusInternalServerError
			}
		} else if rsp.RedirectURL != "" {
			rsp.Status = http.StatusFound
		} else {
			rsp.Status = http.StatusOK
		}
	}

	success := rsp.Status < http.StatusBadRequest

	consoleText := r.RemoteAddr + " [" + strconv.Itoa(rsp.Status) + "]: " + r.Method + " " + r.URL.Path
	if success {
		LogInfo(r, &rsp)
	} else {
		LogError(r, &rsp, errors.Base(consoleText))
	}

	// redirect
	if rsp.Status >= http.StatusMultipleChoices && rsp.Status < http.StatusBadRequest {
		http.Redirect(w, r, rsp.RedirectURL, rsp.Status)
		return
	} else if rsp.RedirectURL != "" {
		LogError(r, &rsp, errors.Base("status code "+strconv.Itoa(rsp.Status)+
			" does not indicate a redirect, but RedirectURL is non-empty '"+
			rsp.RedirectURL+"'"))
	}

	var errorString *string
	if rsp.Error != nil {
		errorStringRaw := rsp.Error.Error()
		errorString = &errorStringRaw
	}

	var trace []string
	if TraceEnabled && errors.HasTrace(rsp.Error) {
		trace = strings.Split(errors.Trace(rsp.Error), "\n")
		for index, element := range trace {
			if strings.HasPrefix(element, "\t") {
				trace[index] = strings.Replace(element, "\t", "    ", 1)
			}
		}
	}

	// http://choly.ca/post/go-json-marshalling/
	jsonResponse, err := json.MarshalIndent(&struct {
		Success bool        `json:"success"`
		Error   *string     `json:"error"`
		Data    interface{} `json:"data"`
		Trace   []string    `json:"_trace,omitempty"`
	}{
		Success: success,
		Error:   errorString,
		Data:    rsp.Data,
		Trace:   trace,
	}, "", "  ")
	if err != nil {
		LogError(r, &rsp, errors.Prefix("Error encoding JSON response: ", err))
	}

	if rsp.Status >= http.StatusInternalServerError {
		LogError(r, &rsp, errors.Prefix(r.Method+" "+r.URL.Path+"\n", rsp.Error))
	}

	w.WriteHeader(rsp.Status)
	w.Write(jsonResponse)
}

// IgnoredFormFields are ignored by FormValues() when checking for extraneous fields
var IgnoredFormFields []string

func FormValues(r *http.Request, params interface{}, validationRules []*v.FieldRules) error {
	ref := reflect.ValueOf(params)
	if !ref.IsValid() || ref.Kind() != reflect.Ptr || ref.Elem().Kind() != reflect.Struct {
		return errors.Err("'params' must be a pointer to a struct")
	}

	structType := ref.Elem().Type()
	structValue := ref.Elem()
	fields := map[string]bool{}
	for i := 0; i < structType.NumField(); i++ {
		name := structType.Field(i).Name
		underscoredName := util.Underscore(name)
		value := strings.TrimSpace(r.FormValue(underscoredName))

		// if param is not set at all, continue
		// comes after call to r.FormValue so form values get parsed internally (if they arent already)
		if len(r.Form[underscoredName]) == 0 {
			continue
		}

		fields[underscoredName] = true
		isPtr := false
		var finalValue reflect.Value

		structField := structValue.FieldByName(name)
		structFieldKind := structField.Kind()
		if structFieldKind == reflect.Ptr {
			isPtr = true
			structFieldKind = structField.Type().Elem().Kind()
		}

		switch structFieldKind {
		case reflect.String:
			finalValue = reflect.ValueOf(value)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if value == "" {
				continue
			}
			castVal, err := cast.ToInt64E(value)
			if err != nil {
				return errors.Err("%s: must be an integer", underscoredName)
			}
			switch structFieldKind {
			case reflect.Int:
				finalValue = reflect.ValueOf(int(castVal))
			case reflect.Int8:
				finalValue = reflect.ValueOf(int8(castVal))
			case reflect.Int16:
				finalValue = reflect.ValueOf(int16(castVal))
			case reflect.Int32:
				finalValue = reflect.ValueOf(int32(castVal))
			case reflect.Int64:
				finalValue = reflect.ValueOf(castVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if value == "" {
				continue
			}
			castVal, err := cast.ToUint64E(value)
			if err != nil {
				return errors.Err("%s: must be an unsigned integer", underscoredName)
			}
			switch structFieldKind {
			case reflect.Uint:
				finalValue = reflect.ValueOf(uint(castVal))
			case reflect.Uint8:
				finalValue = reflect.ValueOf(uint8(castVal))
			case reflect.Uint16:
				finalValue = reflect.ValueOf(uint16(castVal))
			case reflect.Uint32:
				finalValue = reflect.ValueOf(uint32(castVal))
			case reflect.Uint64:
				finalValue = reflect.ValueOf(castVal)
			}
		case reflect.Bool:
			if value == "" {
				continue
			}
			if !validator.IsBoolString(value) {
				return errors.Err("%s: must be one of the following values: %s",
					underscoredName, strings.Join(validator.GetBoolStringValues(), ", "))
			}
			finalValue = reflect.ValueOf(validator.IsTruthy(value))
		default:
			return errors.Err("field %s is an unsupported type", name)
		}

		if isPtr {
			if structField.IsNil() {
				structField.Set(reflect.New(structField.Type().Elem()))
			}
			structField.Elem().Set(finalValue)
		} else {
			structField.Set(finalValue)
		}
	}

	var extraParams []string
	for k := range r.Form {
		if _, ok := fields[k]; !ok && !util.InSlice(k, IgnoredFormFields) {
			extraParams = append(extraParams, k)
		}
	}
	if len(extraParams) > 0 {
		return errors.Err("Extraneous params: " + strings.Join(extraParams, ", "))
	}

	if len(validationRules) > 0 {
		validationErr := v.ValidateStruct(params, validationRules...)
		if validationErr != nil {
			return errors.Err(validationErr)
		}
	}

	return nil
}
