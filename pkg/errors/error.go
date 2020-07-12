package errors

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
)

var errorMapping = map[string]string{
	"Cancelled":          "ValidationError",
	"DeadlineExceeded":   "ValidationError",
	"FailedPrecondition": "ConditionalCheckFailedException",
	"Aborted":            "ValidationError",
}

// Error - this is the error response
type Error struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"message"`
}

// Error - convert error into string
func (e Error) Error() string {
	return e.ErrorCode
}

// New - create new Error
func New(errorCode string, logMessage ...interface{}) *Error {
	err := new(Error)
	err.ErrorCode = errorCode
	err.ErrorMessage = fmt.Sprintln(logMessage...)
	logger.ErrorLogging(err, logMessage)
	return err
}

// HTTPResponse - this is used to set http response
func HTTPResponse(err error, body interface{}) (int, interface{}) {
	e, ok := err.(*Error)
	if ok {
		return http.StatusBadRequest, map[string]interface{}{"code": e.ErrorCode, "message": e.ErrorMessage}
	}
	logger.LogError(err)
	logger.LogErrorF("body: %+v\n ", body)
	return http.StatusInternalServerError, map[string]interface{}{"code": "UncaughtException", "message": err.Error()}
}

// HTTPResponse - this is used to set http response
func (e Error) HTTPResponse(body interface{}) (int, interface{}) {
	logger.LogErrorF("body: %+v\n ", body)

	return http.StatusBadRequest, map[string]interface{}{"code": e.ErrorCode, "message": e.ErrorMessage}
}

// AssignError - this will assign error
func AssignError(err error) *Error {
	if err == nil {
		return nil
	}
	eStr := err.Error()
	for k, v := range errorMapping {
		if strings.Contains(eStr, k) {
			e := new(Error)
			e.ErrorCode = v
			e.ErrorMessage = err.Error()
			logger.ErrorLogging(err)
			return e
		}
	}
	logger.LogDebug(err)
	return nil
}
