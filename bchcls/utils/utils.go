/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package utils contains convenience, helper, and utility functions.
package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("utils")

/*
******************************************************************************************************
Trace functions
******************************************************************************************************
*/

// SetLogLevel sets the log level.
func SetLogLevel(logLevel shim.LoggingLevel) {
	logger.SetLevel(logLevel)
}

// RE_stripFnPreamble uses regex to extract function names (and not the module path).
var RE_stripFnPreamble = regexp.MustCompile(`^.*\.(.*)$`)

// MaxFloat64ToPaddedString is the max absolute value of float64's that can be converted to string.
const MaxFloat64ToPaddedString float64 = 1000000000000

// EnterFnLog logs and returns the current function name at the start of function execution.
// (DEPRECATED) use EnterFnLogger function instead
func EnterFnLog() string {
	fnName := "<unknown>"
	// Skip this function, and fetch the PC and file for its parent
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		fnName = RE_stripFnPreamble.ReplaceAllString(runtime.FuncForPC(pc).Name(), "$1")
	}

	logger.Debugf("---> %s\n", fnName)
	return fnName
}

// ExitFnLog logs the current function name at the end of function execution.
// (DEPRECATED) use ExitFnLogger function instead
func ExitFnLog(s string) {
	logger.Debugf("<--- %s\n", s)
}

// EnterFnLogger logs and returns the current function name at the start of function execution.
func EnterFnLogger(mylogger *shim.ChaincodeLogger) string {
	fnName := "<unknown>"
	// Skip this function, and fetch the PC and file for its parent
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		fnName = RE_stripFnPreamble.ReplaceAllString(runtime.FuncForPC(pc).Name(), "$1")
	}

	mylogger.Debugf("---> %s\n", fnName)
	return fnName
}

// ExitFnLogger logs the current function name at the end of execution.
func ExitFnLogger(mylogger *shim.ChaincodeLogger, s string) {
	mylogger.Debugf("<--- %s\n", s)
}

/*
******************************************************************************************************
Helper functions
******************************************************************************************************
*/

// CheckParam checks if param is in args.
func CheckParam(args []string, param string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == param {
			return true
		}
	}
	return false
}

// InList returns true if item is in listdata, false otherwise.
func InList(listdata []string, item string) bool {
	return CheckParam(listdata, item)
}

// Map applies function f to each element of args.
func Map(args []string, f func(string) string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		val := f(arg)
		if len(val) > 0 {
			result[i] = f(arg)
		}
	}
	return result
}

// GetSetFromList converts a list of strings into a sorted set of strings.
func GetSetFromList(items []string) []string {
	arg := strings.Join(items, ",")
	return GetSet(arg)
}

// GetSet parses a comma-separated string and returns a sorted set of strings.
func GetSet(arg string) []string {
	itemList := Map(strings.Split(arg, ","), strings.TrimSpace)
	itemMap := make(map[string]bool)
	for _, n := range itemList {
		if len(n) > 0 {
			itemMap[n] = true
		}
	}
	items := make([]string, len(itemMap))
	i := 0
	for name := range itemMap {
		items[i] = name
		i++
	}
	sort.Strings(items)
	return items
}

// AddToSet adds an item to a set.
func AddToSet(items []string, item string) []string {
	arg := strings.Join(items, ",")
	arg = arg + "," + item
	return GetSet(arg)
}

// RemoveFromSet removes an item from a set.
func RemoveFromSet(itemList []string, item string) []string {
	itemMap := make(map[string]bool)
	for _, n := range itemList {
		if len(n) > 0 && n != item {
			itemMap[n] = true
		}
	}
	items := make([]string, len(itemMap))
	i := 0
	for name := range itemMap {
		items[i] = name
		i++
	}
	sort.Strings(items)
	return items
}

// FilterSet removes elements from itemList that are not in filterList and returns the result as a set.
func FilterSet(itemList []string, filterList []string) []string {
	itemMap := make(map[string]bool)
	for _, n := range itemList {
		if len(n) == 0 {
			continue
		} else {
			ok := false
			for _, m := range filterList {
				if m == n {
					ok = true
				}
			}
			if ok {
				itemMap[n] = true
			}
		}
	}
	items := make([]string, len(itemMap))
	i := 0
	for name := range itemMap {
		items[i] = name
		i++
	}
	sort.Strings(items)
	return items
}

// FilterOutFromSet removes elements from itemList that are in filterList and returns the result as a set.
func FilterOutFromSet(itemList []string, filterList []string) []string {
	items := itemList
	for _, n := range filterList {
		items = RemoveFromSet(items, n)
	}
	sort.Strings(items)
	return items
}

// GetDataMap expands data and returns a data map.
func GetDataMap(data []string, access []string) map[string]bool {
	datamap := make(map[string]bool)
	for _, item := range data {
		ditem := Map(strings.Split(item, ":"), strings.TrimSpace)
		if len(ditem) == 1 && len(ditem[0]) > 0 {
			if access == nil || len(access) == 0 {
				datamap[ditem[0]] = true
			} else {
				for _, acc := range access {
					datamap[ditem[0]+":"+acc] = true
				}
			}
		} else if len(ditem) >= 2 && len(ditem[0]) > 0 {
			for _, acc := range ditem[1:] {
				if len(acc) > 0 {
					datamap[ditem[0]+":"+acc] = true
				}
			}
		}
	}
	return datamap
}

// NormalizeDataList returns a normalized data list.
func NormalizeDataList(data []string) []string {
	datamap := make(map[string]bool)
	for _, item := range data {
		ditem := Map(strings.Split(item, ":"), strings.TrimSpace)

		if len(ditem) == 1 && len(ditem[0]) > 0 {
			datamap[ditem[0]] = true
		} else if len(ditem) >= 2 && len(ditem[0]) > 0 {
			for _, acc := range ditem[1:] {
				if !datamap[ditem[0]] {
					if len(acc) > 0 {
						datamap[ditem[0]+":"+acc] = true
					}
				}
			}
		}
	}

	datalist2 := GetDataList(datamap)
	datamap2 := make(map[string]bool)
	datalist := []string{}
	for _, d := range datalist2 {
		ditem := Map(strings.Split(d, ":"), strings.TrimSpace)
		if !datamap2[ditem[0]] {
			datalist = append(datalist, d)
			datamap2[d] = true
		}
	}
	sort.Strings(datalist)
	return datalist
}

// GetDataList returns a sorted list of strings from data map.
func GetDataList(data map[string]bool) []string {
	datalist := make([]string, len(data))
	i := 0
	for d := range data {
		datalist[i] = d
		i++
	}
	sort.Strings(datalist)
	return datalist
}

// FilterDataList filters dataList by filter.
func FilterDataList(dataList []string, filter []string) []string {
	filtered := []string{}
	for _, data := range dataList {
		ditem := Map(strings.Split(data, ":"), strings.TrimSpace)
		if len(ditem) == 1 {
			filtered = append(filtered, ditem[0])
		} else if len(ditem) == 2 {
			if CheckParam(filter, ditem[1]) {
				filtered = append(filtered, ditem[0]+":"+ditem[1])
			}
		}
	}
	return filtered
}

// InsertInt inserts value into a sorted list of ints.
func InsertInt(slice []int, value int) []int {
	index := sort.SearchInts(slice, value)
	n := len(slice)
	if index >= n || slice[index] != value {
		if n == cap(slice) {
			newSlice := make([]int, len(slice), 2*len(slice)+1)
			copy(newSlice, slice)
			slice = newSlice
		}
		slice = slice[0 : n+1]
		copy(slice[index+1:], slice[index:])
		slice[index] = value
	}
	return slice
}

// RemoveInt removes value from a sorted list of ints.
func RemoveInt(slice []int, value int) []int {
	index := sort.SearchInts(slice, value)
	if index < len(slice) && slice[index] == value {
		slice = append(slice[:index], slice[index+1:]...)
	}
	return slice
}

// InsertString inserts a value to sorted list of strings.
func InsertString(slice []string, value string) []string {
	i := sort.SearchStrings(slice, value)
	if i == len(slice) {
		slice = append(slice, value)
		return slice
	}
	slice = append(slice, "")
	copy(slice[i+1:], slice[i:])
	slice[i] = value
	return slice
}

// RemoveString removes value from a sorted list of ints.
func RemoveString(slice []string, value string) []string {
	index := sort.SearchStrings(slice, value)
	if index < len(slice) && slice[index] == value {
		slice = append(slice[:index], slice[index+1:]...)
	}
	return slice
}

// PerformHTTP performs an HTTP request and returns the response.
func PerformHTTP(method string, url string, requestBody []byte, headers map[string]string, user []byte, pass []byte, retry ...int) (*http.Response, []byte, error) {
	retryno := 0
	if len(retry) > 0 {
		retryno = retry[0]
	}
	// as a work around to for bad CA certificate date
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}
	client := &http.Client{Transport: transCfg}
	req, err := http.NewRequest(method, url, bytes.NewReader(requestBody))
	if err != nil {
		logger.Errorf("Error building a %s request: %v err: %v", method, url, err)
		return nil, nil, errors.New("Error building a " + method + " request")
	}
	req.ContentLength = int64(len(requestBody))
	//req.Header.Set("Content-Type", "application/json")

	//basic auth
	if user != nil && pass != nil {
		req.SetBasicAuth(string(user[:]), string(pass[:]))
	}

	//header
	if headers != nil {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}

	logger.Debugf("Request: %v", req)

	response, err := client.Do(req)
	if err != nil {
		logger.Errorf("Error attempt to %s %s: %v", method, url, err)
		return nil, nil, errors.New("Error attempt to " + method + " " + url)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("Error reading HTTP resposne body: %v", err)
		return nil, nil, errors.New("Error reading HTTP resposne body")
	}

	// retry one more time if status is 429 Too Many Request
	if response.StatusCode == 429 && retryno < 1 {
		logger.Warningf("Response code: %v for url %v, retrying...", response.StatusCode, url)
		time.Sleep(500 * time.Millisecond)
		return PerformHTTP(method, url, requestBody, headers, user, pass, retryno+1)
	}

	return response, body, nil
}

// leftPad2Len pads the left side of a string (s) to the desired length (totalLen).
// The string is padded with a provided string (padStr).
func leftPad2Len(s string, padStr string, totalLen int) (string, error) {

	// Make sure the padStr is not empty
	if len(padStr) == 0 {
		logger.Error("padStr cannot be an empty string")
		return "", errors.New("padStr cannot be an empty string")
	}

	// Make sure the string is less than totalLen
	if len(s) > totalLen {
		logger.Errorf("\"%v\" is greater than %v digits - padding will fail.", s, totalLen)
		return "", errors.Errorf("\"%v\" is greater than %v digits - padding will fail.", s, totalLen)
	}

	// Figure out how much padding is needed
	numPads := int(math.Ceil(float64(totalLen-len(s)) / float64(len(padStr))))
	retStr := strings.Repeat(padStr, numPads) + s
	return retStr[(len(retStr) - totalLen):], nil
}

// Float64ToPaddedString converts a float64 to an 18-character string padded with "0"s at the front. 4 decimal places are used.
// The number must be less than 12 digits (excluding decimals).
// MaxFloat64ToPaddedString is added to the number so that all negative numbers become positive. Therefore, negative numbers
// will start with a "0" and positive numbers will start with a "1". The actual number will follow.
func Float64ToPaddedString(num float64) (string, error) {
	// Check that the number is small enough (less than 12 digits)
	if math.Abs(num) >= MaxFloat64ToPaddedString {
		logger.Errorf("To convert to padded string, abs(%v) must be less than %v", num, MaxFloat64ToPaddedString)
		return "", errors.Errorf("To convert to padded string, abs(%v) must be less than %v", num, MaxFloat64ToPaddedString)
	}
	// Add MaxFloat64ToPaddedString to make all numbers positive and pad with 0's
	return leftPad2Len(strconv.FormatFloat(num+MaxFloat64ToPaddedString, 'f', 4, 64), "0", 18)
}

// TimestampToDateString converts a timestamp to an EST date string.
func TimestampToDateString(ts int64) string {
	tm := time.Unix(ts, 0)
	loc, _ := time.LoadLocation("EST")
	return tm.In(loc).String()
}

// DateStringToTimestamp converts a date string to a timestamp.
func DateStringToTimestamp(date string) (int64, error) {
	layouts := []string{"2006-01-02 15:04:05 -0700 MST", time.ANSIC, time.UnixDate, time.RFC822, time.RFC822Z, time.RFC850, time.RFC1123, time.RFC1123Z, time.RFC3339, "01/02/2006"}
	for _, layout := range layouts {
		ts, err := time.Parse(layout, date)
		if err == nil {
			return ts.Unix(), nil
		}
	}
	return 0, errors.New("Failed to convert")
}

// ToQueryString returns a map of data into query parameters.
func ToQueryString(m map[string][]string) string {
	params := url.Values{}
	for k, vl := range m {
		for _, v := range vl {
			params.Add(k, v)
		}
	}
	return params.Encode()
}

// RemoveItemFromList returns list with the first element that matches item removed.
func RemoveItemFromList(list []string, item string) []string {
	for i, v := range list {
		if v == item {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
	return list
}

// IsStringEmpty returns true if the provided string is nil or empty, false otherwise.
func IsStringEmpty(s interface{}) bool {
	switch t := s.(type) {
	case string:
		return s == nil || len(strings.TrimSpace(s.(string))) == 0
	default:
		logger.Errorf("Invalid data type passed to IsStringEmpty: %T", t)
		return true
	}
}

// IsInstanceOf returns true if object and objectType are of the same type, false otherwise.
func IsInstanceOf(object, objectType interface{}) bool {
	return reflect.TypeOf(object) == reflect.TypeOf(objectType)
}

// ConvertToString determines the type of val and returns it as a string.
// Valid types for val are bool, int, int64, float64, or string.
// If val is an int, int64, or float64, its absolute value cannot exceed utils.MaxFloat64ToPaddedString.
// This function is not to be used for general conversion to string, as the strings produced by this function may not look at all like the input.
// Use this function to produce strings that are suitable for querying assets by index.
func ConvertToString(val interface{}) (string, error) {
	logger.Debugf("Converting val: \"%v\" of type: \"%T\" to a string", val, val)
	switch v := val.(type) {
	case bool:
		return strconv.FormatBool(v), nil
	case int64:
		return Float64ToPaddedString(float64(v))
	case int:
		return Float64ToPaddedString(float64(v))
	case float64:
		return Float64ToPaddedString(v)
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		return "", errors.Errorf("Cannot convert type \"%T\" to string", v)
	}
}

// EqualStringArrays returns if two string arrays are the same.
func EqualStringArrays(val1 []string, val2 []string) bool {
	str1 := strings.Join(val1, "_,_")
	str2 := strings.Join(val2, "_,_")
	return str1 == str2
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}
