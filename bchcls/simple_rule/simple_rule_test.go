/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package simple_rule

import (
	"common/bchcls/test_utils"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func TestSample(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestSample")

	/*

	   simple_rule_engine

	   ------------------------------------------
	   1. Rule expression format:

	   Rule expressions are always in the following format:

	   in JSON:
	   { "operator" : [ arg1, arg2, ... ] }

	   for example:
	   { "+" : [1, 2.4, {"-", [30, 10]}]}

	   The value of args can be any final value (number, string, boolean, array, or map),
	   or it can be a rule expression.

	   In Go, its type is:
	   map[string]interface{}

	   The above example can be implemented as follows:
	   r1 := make(map[string]interface{})
	   r1["-"] = []interface{}{30,10}
	   r := make(map[string]interface{})
	   r["+"] = []interface{}{ 1, 2.4, 6, r1 }

	   There is an util function, R(), to create a rule:
	   R("operator", arg1, arg2, ...)

	   The above rule expression can be implemented as follows, using R function:
	   r := R("+", 1, 2.4, R("-", 20,10))

	   Note that rule expressions are only evaluted when you call apply() function,
	   and rule_expression is never evaluated during creation.

	   All numeric data in rule_expression are treated as float64 values.
	   However, if an integer value is required (e.g. % operator), it checks
	   if the value is a whole number and converts it to an int64 value for the internal
	   computation. If the value is not a whole number when an integer is required, an error will be
	   retuned.

	   For example,
	   {"/": [5, 2]} will be evaluated to 2.5
	   {"/": [5.0, 2]} will be evaluated to 2.5
	   {"/": [5, 2.0]} will be evaluated to 2.5
	   {"/": [5.0, 2.0]} will be evaluated to 2.5

	   {"%": [5, 2]} will be evaluated to 1.0
	   {"%": [5.0, 2]} will be evaluated to 1.0
	   {"%": [5, 2.0]} will be evaluated to 1.0
	   {"%": [5.0, 2.0]} will be evaluated to 1.0

	   {"%": [5.2, 2.0]} will be throw an error since 5.2 is not a whole(int) number.



	   ------------------------------------------
	   2. Data map

	   A Rule is always applied with data.
	   In JSON, the data is always a map.
	   {
	   	"variable_name1": value1,
	   	"variable_name2": value2,
	   	...
	   }

	   In Go, data's type is also map[string]interface{}, like a rule.
	   However, the value of variable should be the final value and not a rule expression.

	   value can be a number, string, boolean, array (list) or map (dict).
	   For example:
	   { "name": "Tom", "age": 24, "hobby": ["movies", "coding"] }

	   Just like the rule_expression, all numeric data in a data map are treated as float64 values.

	   A variable name cannot start with "$".
	   Variables that start with "$" are are reserved for internal use only.
	   e.g.  "$result" is reserved for the return value.

	   There are two util functions for creating a data map: D() and M()

	   D(value)
	   D function returns normalized data as an interface{}.

	   D(1) returns interface{}(float64(1))

	   If you supply more than one value to the D() function, it returns normalized []interface{}
	   e.g.
	   D(1, "a", "b") returns []interface{}{float64(1), "a", "b"}

	   M( value1, value2, ... )
	   returns map[string]interface{}

	   value1, value2 are a list of values with a []interface{ key, value } format.

	   Fsor example, using M() and D() functions, you can create data map as follows:

	   M( D("name", "Tom"), D("age", 32), D("sex", "male") )

	   returns the equivalent of the following JSON:

	   { "name": "Tom", "age": 32, "sex": "male" }

	   The data map is accessed by the "var" operator.
	   {"var": ["variable_name", "key or index" ... ]}

	   "key or index"  is used to drill down into the objects and arrays.
	   "var" operator examples are listed in TestVar section below.

	   ------------------------------------------
	   3. Create a rule with the NewRule() function.

	   A new rule is always created using NewRule() function.

	   rule := NewRule(rule_expr, init_data)

	   - rule_expr is the rule expression format explained above.
	     Value of rule_expr can be either a JSON string or a map[string]interface{} value.
	   - init_data is an optional data parameter to be specified when a rule is created.
	     Value of init_value can also be either a JSON string or a map[string]interface{} value.

	   Data map should be supplied when you actually evaluate the rule.
	   However, you can also optionally define the initial data when you are creating a rule,
	   and this init value is merged with the data that you supplied for the actual evaluation.

	   value of rule_expr and init_data can be either a JSON string or a map[string]interface{}.
	   Or you can use the R function to create a rule expression.

	   The following three examples are equivalent, and they generate the exact same rule.
	*/

	var r1 = NewRule(`{"+": [1, 2.4, 6]}`)
	m1, e1 := r1.Apply(nil)
	logger.Debugf("r1: %v %v %v", r1.GetExprJSON(), m1, e1)

	var e = make(map[string]interface{})
	e["+"] = []interface{}{1, 2.4, 6}
	var r2 = NewRule(e)
	m2, e2 := r2.Apply(nil)
	logger.Debugf("r2: %v %v %v", r2.GetExprJSON(), m2, e2)

	var r3 = NewRule(R("+", 1, 2.4, 6))
	m3, e3 := r3.Apply(nil)
	logger.Debugf("r3: %v %v %v", r3.GetExprJSON(), m3, e3)

	/*

	   Rule has the following methods:

	   GetExpr() - returns rule expression as interface{}
	   example:
	*/
	var expr interface{} = r3.GetExpr()
	logger.Debugf("expr: %v", expr)

	/*
	   GetExprJSON() - returns rule expression as JSON string
	   example:
	*/
	var exprJ string = r3.GetExprJSON()
	logger.Debugf("expr JSON: %v", exprJ)

	/*
	   GetInit() - returns init data as map[string]interface{}
	   example:
	*/
	var init map[string]interface{} = r3.GetInit()
	logger.Debugf("init: %v", init)
	/*
	   GetInit() - returns init data as JSON string
	   example:
	*/
	var initJ string = r3.GetInitJSON()
	logger.Debugf("init JSON: %v", initJ)
	/*
	   Apply(data) - data is map[string]interface{} or JSON string or nil if data is not needed
	                 and it returns
	                 result data map as map[string]interface{}, and an error
	                 the evaluated value is stored with "$result" key of the result data map

	   Also, there is
	   ToJSON( data ) util function with returns JSON string of data

	   example:
	*/
	result, err := r3.Apply(nil)
	logger.Debugf("result: %v ;error: %v", result, err)
	logger.Debugf("result JSON: %v", ToJSON(result))

	/*


	   ------------------------------------------
	   4. Supported operators


	   (Math operations)
	   +
	   -
	   *
	   /
	   %
	   max
	   min
	   int
	   float
	   round  (round toward 0)
	   floor
	   ceil
	   sqrt


	   (String operations)
	   +  (same as cat)
	   cat
	   max
	   min
	   int
	   float
	   contains
	   in
	   substr


	   (Logical, Boolean and Comparison operations)
	   ==
	   ===
	   !=
	   !==
	   >
	   >=
	   <
	   <=
	   between (using < and <=)
	   !  (same as not)
	   not
	   !!  (same as bool)
	   bool
	   and
	   &&  (same as and)
	   or
	   ||  (same as or)
	   if


	   (Array and Dictionary operations)
	   array
	   list  (same as array)
	   in
	   for_each
	   map  (same as for_each)
	   filter
	   reduce
	   merge
	   dict
	   len
	   keys
	   has_key
	   get


	   (Accessing Data operations)
	   var
	   missing
	   missing_some
	   :=  (same as let)
	   let
	   proc

	   (Miscellaneous)
	   log

	*/

	test_utils.AssertTrue(t, true, "ok")
}

func TestMath(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestMath")

	//JSON
	r := NewRule(`{"+": [1, 2, 3.5]}`)
	m, e := r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(6.5), "ok")

	r = NewRule(R("+", 1, 2, 3.5))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(6.5), "ok")

	//error
	r = NewRule(R("+", 1, "a"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("-", 10, 1, 2.2))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(6.8), "ok")

	//single value
	r = NewRule(R("-", 10))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-10), "ok")

	//single value
	r = NewRule(`{"-": 10}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-10), "ok")

	//error
	r = NewRule(R("-", 1, "a"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("*", 2, 3, 4))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(24), "ok")

	//stops if value becomes 0, no error
	r = NewRule(R("*", 2, 3, 0, "a"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(0), "ok")

	r = NewRule(R("/", 10, 2, 2))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(2.5), "ok")

	//div by zero error
	r = NewRule(R("/", 2, 3, 0, 1))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("%", 10, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(1), "ok")

	//error non integer
	r = NewRule(R("%", 10, 2.5))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("max", 10, 3, -1, 25.3, 4))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(25.3), "ok")

	//error not number
	r = NewRule(R("max", 10, 2.5, "a", 55))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("min", 10, 3, -1, 25.3, 4))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-1), "ok")

	r = NewRule(R("int", 10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	//string
	r = NewRule(R("int", "10.8"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	//error
	r = NewRule(R("int", "abc"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("float", int64(10)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"].(float64) == float64(10), "ok")

	//string
	r = NewRule(R("float", "10.8"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10.8), "ok")

	//error
	r = NewRule(R("float", "abc"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("round", 10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(11), "ok")

	r = NewRule(R("round", 10.5))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(11), "ok")

	r = NewRule(R("round", 10.4))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	r = NewRule(R("round", -10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-11), "ok")

	r = NewRule(R("round", -10.5))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-11), "ok")

	r = NewRule(R("round", -10.3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-10), "ok")

	r = NewRule(R("floor", 10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	r = NewRule(R("floor", 10.4))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	r = NewRule(R("floor", -10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-11), "ok")

	r = NewRule(R("floor", -10.3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-11), "ok")

	r = NewRule(R("ceil", 10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(11), "ok")

	r = NewRule(R("ceil", 10.4))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(11), "ok")

	r = NewRule(R("ceil", -10.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-10), "ok")

	r = NewRule(R("ceil", -10.3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-10), "ok")

	r = NewRule(R("sqrt", 16))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(4), "ok")

	//error
	r = NewRule(R("sqrt", -16))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")
}

func TestString(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestString")

	r := NewRule(R("cat", "My name is ", "Tom ", "Sayer"))
	m, e := r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("My name is Tom Sayer"), "ok")

	r = NewRule(R("max", "orange", "apple", "grape"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("orange"), "ok")

	r = NewRule(R("min", "orange", "apple", "grape"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("apple"), "ok")

	r = NewRule(R("int", "10.8"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	r = NewRule(R("float", "10.8"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10.8), "ok")

	r = NewRule(R("contains", "My name is Tom", "Tom"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("contains", "My name is Tom", "Jane"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("in", "Tom", "My name is Tom"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("in", "Jane", "My name is Tom"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("substr", "abcdefghijk", 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("defghijk"), "ok")

	r = NewRule(R("substr", "abcdefghijk", -3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("ijk"), "ok")

	r = NewRule(R("substr", "abcdefghijk", 3, 8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("defghijk"), "ok")

	r = NewRule(R("substr", "abcdefghijk", 3, -3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("defgh"), "ok")

	//index out of range
	r = NewRule(R("substr", "abcdefghijk", 3, 30))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("defghijk"), "ok")
}

func TestBool(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestBool")

	r := NewRule(R("==", 10, 10.0))
	m, e := r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("==", "tom", "tom"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("==", 1, "1"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("==", 0, false))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("==", false, false))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("==", 10, 12))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("==", 10, "a"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("===", 1, 1))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("===", 1, "1"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	//error
	r = NewRule(R("==", 10, "a", "b"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R("!=", 10, 10.0))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("!=", "tom", "tom"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("!=", false, false))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("!=", 10, 12))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("!=", 10, "a"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("!=", 1, "1"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("!==", 1, 2))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("!==", 1, "1"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{">": [10, 9.8]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R(">", 10, 9.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R(">", "xyx", "abc"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	//error
	r = NewRule(R(">", "xyx", 10))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, e != nil, "ok")

	r = NewRule(R(">", 9.8, 10))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R(">=", 10, 9.8))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("<", 9.8, 10))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("<=", 9.8, 10))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("<", 1, 2, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("<", 1, 1, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("<", 1, 4, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("<=", 1, 2, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("<=", 1, 1, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("<=", 1, 4, 3))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{ "<": [0, {"var":"temp"}, 100]}`)
	m, e = r.Apply(`{"temp" : 37}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("!", true))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("not", true))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("!", R("<", 10, 9.8)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"!!": [ [] ] }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"!!": ["0"] }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"bool": ["0"] }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("and", true, true))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("&&", true, false))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(R("or", false, true))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("||", true, false))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("if", R(">", 20, 10), "bigger", "smaller"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("bigger"), "ok")

	r = NewRule(R("if",
		R(">", 2, 10), "big",
		R(">", 2, 5), "mid",
		R(">", 2, 1), "small",
		"very small"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("small"), "ok")

}

func TestArray(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestArray")

	//note that {"+" [1,2]} is not evaluated
	r := NewRule(R("array", 1, 2, R("+", 1, 2), 4.0, "my val", 1, "test"))
	m, e := r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, len(m["$result"].([]interface{})) == 7, "ok")

	r = NewRule(R("list", 1, 2, R("+", 1, 2), 4.0, "my val", 1, "test"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, len(m["$result"].([]interface{})) == 7, "ok")

	//note that you can just use list value with out using the rule, too
	r = NewRule(`[ 1, 2, {"+": [1, 2]}, 4.0, "my val", 1, "test"]`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, len(m["$result"].([]interface{})) == 7, "ok")

	r = NewRule(R("in", "my val", R("array", 1, 2, R("+", 1, 2), 4.0, "my val", 1, "test")))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	//get array from data
	d := M(D("my list", D(1, 2, 3, 4.0, "my val", 1, "test")))
	r = NewRule(R("in", "my val", R("var", "my list")), d)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	// {"+", 1,2} evaluated in runtime
	r = NewRule(R("in", 3, R("array", 1, 2, R("+", 1, 2), 4.0, "my val", 1, "test")))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(R("in", "your val", R("array", 1, 2, R("+", 1, 2), 4.0, "my val", 1, "test")))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	//note that {"var": ["$current"]} refers to the current value
	//also, note that folowing are all equivalent expressions, and they refers to the current value
	//{"var": ["$current"]}, {"var": "$current"} , {"var": [""]}, {"var": ""}
	r = NewRule(R("map", R("array", 1, 2, 3, 4), R("+", R("var", "$current"), 2)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(3, 4, 5, 6)), "ok")

	//note that {"var": ["$current"]} refers to the current value
	//r = NewRule( `{"for_each": [{"array": [1,2,3,4]}, {"+": [{"var": ["$current"]}, 2]}]}` )
	r = NewRule(`{"for_each": [[1,2,3,4], {"+": [{"var": ["$current"]}, 2]}]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(3, 4, 5, 6)), "ok")

	r = NewRule(`{"for_each": [[1,2,3,4], {"+": [{"var": ""}, 2]}]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(3, 4, 5, 6)), "ok")

	//note that {"var": ["$current"]} refers to the current value
	r = NewRule(R("filter", R("array", 1, 2, 3, 4), R(">", R("var", "$current"), 2)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(3, 4)), "ok")

	//adding all elements in the array using reduce
	//note that {"var": ["$current"]} refers to the current value
	//note that {"var": ["$accumulator"]} refers to the reduced accumulator value
	//note that 0 is the initial $acculator value
	r = NewRule(R("reduce",
		R("array", 1, 2, 3, 4, 5),
		R("+",
			R("var", "$current"),
			R("var", "$accumulator")),
		0))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(15), "ok")

	//same as above using JSON
	r = NewRule(`{
					"reduce":
						[
							{"array":[1,2,3,4,5]},
							{"+":
								[
									{"var":["$current"]},
									{"var":["$accumulator"]}
								]
							},
							0
						]
				  }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(15), "ok")

	//merge two arrays
	//note that elements of input arrays are not evaluated
	r = NewRule(R("merge", R("array", "a", "b"), R("array", "c", "d"), R("array", "e", 10.0)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D("a", "b", "c", "d", "e", 10)), "ok")

	//create map (dictionary)
	r = NewRule(R("dict", D("a", 1), D("b", 2), D("c", 3)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, len(m["$result"].(map[string]interface{})) == 3, "ok")

	//length of array
	r = NewRule(R("len", R("merge", R("array", "a", "b"), R("array", "c", "d"), R("array", "e", 10.0))))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(6), "ok")

	//length of map
	r = NewRule(`{"len": [{"a":1, "b":2, "c":3}]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(3), "ok")

	r = NewRule(R("has_key", "a", R("dict", D("a", 1), D("b", 2), D("c", 3))))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	//get from array
	r = NewRule(R("get", 3, R("merge", R("array", "a", "b"), R("array", "c", "d"), R("array", "e", 10.0))))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("d"), "ok")

	//get from map
	r = NewRule(R("get", "b", R("dict", D("a", 1), D("b", 2), D("c", 3))))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(2), "ok")

	r = NewRule(`{"all" : [ [1,2,3], {">":[{"var":""}, 0]} ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"some" : [ [-1,0,1], {">":[{"var":""}, 0]} ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"none" : [ [-3,-2,-1], {">":[{"var":""}, 0]} ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	d1 := `{"pies":[
			  {"filling":"pumpkin","temp":110},
			  {"filling":"rhubarb","temp":210},
			  {"filling":"apple","temp":310}
			]}`
	r = NewRule(`{"some" : [ {"var":"pies"}, {"==":[{"var":"filling"}, "apple"]} ]}`)
	m, e = r.Apply(d1)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")
}

func TestVar(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVar")

	d := `{
			"hobby": ["movie", "game", "surfing"],
			"name": {"last": "Sawyer", "first": "Tom"},
			"age": 45,
			"address": {"street": "75 Binney St", "residents": ["Jane", "John", "Tom"]},
			"job": [ {"company": "IBM", "year": 2018}, {"company": "Watson", "year":2017} ]
		  }`
	r := NewRule(R("var", "age"))
	m, e := r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(45), "ok")

	// adding data during rule creating
	r = NewRule(R("var", "age"), d)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(45), "ok")

	// var from array
	r = NewRule(R("var", "hobby.1"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("game"), "ok")

	// var drill down; array, map
	r = NewRule(R("var", "job.1.company"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("Watson"), "ok")

	// var from map
	r = NewRule(R("var", "name.first"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("Tom"), "ok")

	// var drill down; map, array
	r = NewRule(R("var", "address.residents.0"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("Jane"), "ok")

	// var non-existing field; addres.new_address does not exist
	r = NewRule(R("var", "address.new_address"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(nil), "ok")

	r = NewRule(`{"bool": [{"var":"address.new_address"}]}`)
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	// missing
	r = NewRule(`{"missing":["a", "b"]}`)
	m, e = r.Apply(`{"a":"apple", "c":"carrot"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{"b"}), "ok")

	r = NewRule(`{"missing":["a", "b"]}`)
	m, e = r.Apply(`{"a":"apple", "b":"banana"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{}), "ok")

	//missing some
	r = NewRule(`{"missing_some":[1, ["a", "b", "c"]]}`)
	m, e = r.Apply(`{"a":"apple"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{}), "ok")

	r = NewRule(`{"missing_some":[2, ["a", "b", "c"]]}`)
	m, e = r.Apply(`{"a":"apple"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D("b", "c")), "ok")

	//assign variable, return value is the value of the variable
	r = NewRule(R("let", "address", "75 Binney St. Cambridge, MA, USA"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["address"] == D("75 Binney St. Cambridge, MA, USA"), "ok")

	r = NewRule(R(":=", "company", "IBM"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["company"] == D("IBM"), "ok")

	// replace whole value
	r = NewRule(R(":=", "hobby", D("read", "write")))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["hobby"], D("read", "write")), "ok")

	//append array value, index is -1
	r = NewRule(R(":=", "hobby", -1, "SNS"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["hobby"], D("movie", "game", "surfing", "SNS")), "ok")

	//replace array value
	r = NewRule(R(":=", "hobby", 2, "SNS"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["hobby"], D("movie", "game", "SNS")), "ok")

	//add or update map value
	r = NewRule(R(":=", "name", "middle", "Alex"))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["name"].(map[string]interface{})["middle"] == D("Alex"), "ok")

	//proce, process multiple rules, the return value is the last rule's return value
	//any rules can be processed; not just :=
	// {"proc": [ 	{":=": ["name", "middle", "Alex"]},
	//		{":=": ["hobby", -1, "SNS"]}
	//	     ]}

	r = NewRule(R("proc",
		R(":=", "name", "middle", "Alex"),
		R(":=", "hobby", -1, "SNS")))
	m, e = r.Apply(d)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["name"].(map[string]interface{})["middle"] == D("Alex") && reflect.DeepEqual(m["hobby"], D("movie", "game", "surfing", "SNS")), "ok")

	// constant string value
	r = NewRule("ABC")
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("ABC"), "ok")

	// constant numeric value
	r = NewRule(123.5)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(123.5), "ok")

	// constant bool value
	r = NewRule(true)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	// constant list value
	r = NewRule(D(1, 2, "a", "b"))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(1, 2, "a", "b")), "ok")

	r = NewRule(`[1,2,"a","b"]`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(1, 2, "a", "b")), "ok")

	// constnat map value
	r = NewRule(M(D("a", 1), D("b", 2)))
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], M(D("a", 1), D("b", 2))), "ok")

	r = NewRule(`{"a":1, "b":2}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], M(D("a", 1), D("b", 2))), "ok")

	//log
	r = NewRule(`{"log": "apple"}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"].(string) == "apple", "ok")

}

//various other rule tests
func TestRule(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestRule")

	//complex example
	//find which one of my hobbies are good hobbies.
	//also, find if I am a young person or not
	d0 := `{
		      "max_age": 30,
		      "good_hobbies": [ "sleep", "movie", "eat"]
		    }`

	d1 := `{
		      "person": {
		          "hobby": ["movie", "game", "sleep", "surfing"],
		          "name": {"last": "Sawyer", "first": "Tom"},
		          "age": 45,
		          "smoke": false
		      }
		    }`

	d2 := `{
		      "person": {
		          "hobby": ["sleep", "drive", "study"],
		          "name": {"last": "Doe", "first": "John"},
		          "age": 20,
		          "smoke": false
		      }
		    }`

	rule_exp := `{ "proc": 
		          [
		            {":=": [
		                  "my_name",
		                  {"cat":
		                    [ 
		                      {"get": ["first", {"var": ["person.name"]}]},
		                      " ",
		                      {"get": ["last", {"var": ["person.name"]}]}
		                    ]
		                    
		                  }
		                ]
		            },
		            
		            {"if":   [
		                  {">=":   [{"var": ["max_age"]}, {"var": ["person.age"]}]},
		                    {":=": ["am_i_young_person", true]},
		                    {":=": ["am_i_young_person", false]}
		                ]
		            },
		            
		            {":=":   [ 
		                  "my_good_hobbies",
		                  {"filter":   
		                    [
		                      {"var": ["person", "hobby"]},
		                      {"in":  [
		                            {"var": ["$current"]},
		                            {"var": ["good_hobbies"]}
		                          ]
		                      }
		                    ]
		                  }
		                ]
		            }
		          ]
		        }`

	logger.Debug("----- rule")

	r := NewRule(rule_exp, d0)
	logger.Debugf("rule exp: %v", r.GetExprJSON())
	logger.Debugf("init data: %v", r.GetInitJSON())

	logger.Debug("----- apply to person 1")
	m, e := r.Apply(d1)
	logger.Debugf("return value: %v %v", ToJSON(m), e)
	logger.Debugf("my name: %v", m["my_name"])
	logger.Debugf("my good hobbies: %v", m["my_good_hobbies"])
	logger.Debugf("am i young person: %v", m["am_i_young_person"])
	test_utils.AssertTrue(t, m["am_i_young_person"] == D(false), "ok")

	logger.Debug("----- apply to person 2")
	m, e = r.Apply(d2)
	logger.Debugf("return value: %v %v", ToJSON(m), e)
	logger.Debugf("my name: %v", m["my_name"])
	logger.Debugf("my good hobbies: %v", m["my_good_hobbies"])
	logger.Debugf("am i young person: %v", m["am_i_young_person"])
	test_utils.AssertTrue(t, m["am_i_young_person"] == D(true), "ok")

	rule_exp = `{"if" :[
				    {"merge": [
				      {"missing":["first_name", "last_name"]},
				      {"missing_some":[1, ["cell_phone", "home_phone"] ]}
				    ]},
				    "We require first name, last name, and one phone number.",
				    "OK to proceed"
				  ]}`

	r = NewRule(rule_exp)
	m, e = r.Apply(`{"first_name":"Bruce", "last_name":"Wayne"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("We require first name, last name, and one phone number."), "ok")

	r = NewRule(`{"while": [ 
		{"<": [{"var": "index"}, 10]}, 
		{"proc": [
			{"log": [{"var": "index"}]},
			{":=": [ "index", {"+": [{"var": "index"}, 1]} ]}
		]}
	]}`)
	m, e = r.Apply(`{"index":1}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	//nil test
	r = NewRule(nil)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == nil, "ok")

	//data test
	mydata := make(map[string]string)
	mydata["a"] = "b"
	mydata["b"] = "a"
	mydata2, _ := json.Marshal(&mydata)
	r = NewRule(`{">": [ {"var": "a"}, {"var": "b"}]}`)
	m, e = r.Apply(string(mydata2))
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

}

//JsonLogic compatibility test
func TestJL(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestJL")

	r := NewRule(`{ "var" : ["a"] }`)
	m, e := r.Apply(`{ "a":1, "b":2 }`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(1), "ok")

	r = NewRule(`{ "var":"a" }`)
	m, e = r.Apply(`{ "a":1, "b":2 }`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(1), "ok")

	r = NewRule(`{"var":["z", 26]}`)
	m, e = r.Apply(`{"a":1,"b":2}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(26), "ok")

	r = NewRule(`{"var" : "champ.name"}`)
	m, e = r.Apply(`{
	  "champ" : {
	    "name" : "Fezzig",
	    "height" : 223
	  },
	  "challenger" : {
	    "name" : "Dread Pirate Roberts",
	    "height" : 183
	  }
	}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("Fezzig"), "ok")

	r = NewRule(`{"var":1}`)
	m, e = r.Apply(`["zero", "one", "two"]`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("one"), "ok")

	r = NewRule(`{ "and" : [
	  {"<" : [ { "var" : "temp" }, 110 ]},
	  {"==" : [ { "var" : "pie.filling" }, "apple" ] }
	] }`)
	m, e = r.Apply(`{
	  "temp" : 100,
	  "pie" : { "filling" : "apple" }
	}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{ "cat" : [
	    "Hello, ",
	    {"var":""}
	] }`)
	m, e = r.Apply("Dolly")
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("Hello, Dolly"), "ok")

	r = NewRule(`{"missing":["a", "b"]}`)
	m, e = r.Apply(`{"a":"apple", "c":"carrot"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{"b"}), "ok")

	r = NewRule(`{"missing":["a", "b"]}`)
	m, e = r.Apply(`{"a":"apple", "b":"banana"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{}), "ok")

	r = NewRule(`{"if":[
	  {"missing":["a", "b"]},
	  "Not enough fruit",
	  "OK to proceed"
	]}`)
	m, e = r.Apply(`{"a":"apple", "b":"banana"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("OK to proceed"), "ok")

	r = NewRule(`{"missing_some":[1, ["a", "b", "c"]]}`)
	m, e = r.Apply(`{"a":"apple"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{}), "ok")

	r = NewRule(`{"missing_some":[2, ["a", "b", "c"]]}`)
	m, e = r.Apply(`{"a":"apple"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], []interface{}{"b", "c"}), "ok")

	r = NewRule(`{"if" :[
	    {"merge": [
	      {"missing":["first_name", "last_name"]},
	      {"missing_some":[1, ["cell_phone", "home_phone"] ]}
	    ]},
	    "We require first name, last name, and one phone number.",
	    "OK to proceed"
	  ]}`)
	m, e = r.Apply(`{"first_name":"Bruce", "last_name":"Wayne"}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("We require first name, last name, and one phone number."), "ok")

	r = NewRule(`{"if" : [ true, "yes", "no" ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("yes"), "ok")

	r = NewRule(`{"if" : [ false, "yes", "no" ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("no"), "ok")

	r = NewRule(`{"if" : [
	  {"<": [{"var":"temp"}, 0] }, "freezing",
	  {"<": [{"var":"temp"}, 100] }, "liquid",
	  "gas"
	]}`)
	m, e = r.Apply(`{"temp":55}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("liquid"), "ok")

	r = NewRule(`{"==" : [1, 1]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"==" : [1, "1"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"==" : [0, false]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"===" : [1, 1]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"===" : [1, "1"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"!=" : [1, 2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"!=" : [1, "1"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"!==" : [1, 2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"!==" : [1, "1"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"!": [true]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"!": true}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"!!": [ [] ] }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"or": [true, false]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"or":[false, true]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"or":[false, "a"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("a"), "ok")

	r = NewRule(`{"or":[false, 0, "a"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("a"), "ok")

	r = NewRule(`{"and": [true, true]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"and": [true, false]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"and":[true,"a",3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(3), "ok")

	r = NewRule(`{"and": [true,"",3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(""), "ok")

	r = NewRule(`{">" : [2, 1]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{">=" : [1, 1]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"<" : [1, 2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"<=" : [1, 1]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"<" : [1, 2, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"<" : [1, 1, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"<" : [1, 4, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{"<=" : [1, 2, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"<=" : [1, 1, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"<=" : [1, 4, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(false), "ok")

	r = NewRule(`{ "<": [0, {"var":"temp"}, 100]}`)
	m, e = r.Apply(`{"temp" : 37}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"max":[1,2,3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(3), "ok")

	r = NewRule(`{"min":[1,2,3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(1), "ok")

	r = NewRule(`{"+":[4,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(6), "ok")

	r = NewRule(`{"-":[4,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(2), "ok")

	r = NewRule(`{"*":[4,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(8), "ok")

	r = NewRule(`{"/":[4,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(2), "ok")

	r = NewRule(`{"+":[2,2,2,2,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(10), "ok")

	r = NewRule(`{"*":[2,2,2,2,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(32), "ok")

	r = NewRule(`{"-": 2 }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(-2), "ok")

	r = NewRule(`{"-": -2 }`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(2), "ok")

	r = NewRule(`{"+" : "3.14"}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(3.14), "ok")

	r = NewRule(`{"%": [101,2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(1), "ok")

	r = NewRule(`{"map":[
	  {"var":"integers"},
	  {"*":[{"var":""},2]}
	]}`)
	m, e = r.Apply(`{"integers":[1,2,3,4,5]}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(2, 4, 6, 8, 10)), "ok")

	r = NewRule(`{"filter":[
	  {"var":"integers"},
	  {"%":[{"var":""},2]}
	]}`)
	m, e = r.Apply(`{"integers":[1,2,3,4,5]}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(1, 3, 5)), "ok")

	r = NewRule(`{"reduce":[
	    {"var":"integers"},
	    {"+":[{"var":"current"}, {"var":"accumulator"}]},
	    0
	]}`)
	m, e = r.Apply(`{"integers":[1,2,3,4,5]}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(15), "ok")

	r = NewRule(`{"all" : [ [1,2,3], {">":[{"var":""}, 0]} ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"some" : [ [-1,0,1], {">":[{"var":""}, 0]} ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"none" : [ [-3,-2,-1], {">":[{"var":""}, 0]} ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"some" : [ {"var":"pies"}, {"==":[{"var":"filling"}, "apple"]} ]}`)
	m, e = r.Apply(`{"pies":[
	  {"filling":"pumpkin","temp":110},
	  {"filling":"rhubarb","temp":210},
	  {"filling":"apple","temp":310}
	]}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"merge":[ [1,2], [3,4] ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(1, 2, 3, 4)), "ok")

	r = NewRule(`{"merge":[ 1, 2, [3,4] ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D(1, 2, 3, 4)), "ok")

	r = NewRule(`{"missing" :
	  { "merge" : [
	    "vin",
	    {"if": [{"var":"financing"}, ["apr", "term"], [] ]}
	  ]}
	}`)
	m, e = r.Apply(`{"financing":true}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], D("vin", "apr", "term")), "ok")

	r = NewRule(`{"missing" :
	  { "merge" : [
	    "vin",
	    {"if": [{"var":"financing"}, ["apr", "term"], [] ]}
	  ]}
	}`)
	m, e = r.Apply(`{"financing":false}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, reflect.DeepEqual(m["$result"], A("vin")), "ok")

	r = NewRule(`{"in":[ "Ringo", ["John", "Paul", "George", "Ringo"] ]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"in":["Spring", "Springfield"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D(true), "ok")

	r = NewRule(`{"cat": ["I love", " pie"]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("I love pie"), "ok")

	r = NewRule(`{"cat": ["I love ", {"var":"filling"}, " pie"]}`)
	m, e = r.Apply(`{"filling":"apple", "temp":110}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("I love apple pie"), "ok")

	r = NewRule(`{"substr": ["jsonlogic", 4]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("logic"), "ok")

	r = NewRule(`{"substr": ["jsonlogic", -5]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("logic"), "ok")

	r = NewRule(`{"substr": ["jsonlogic", 1, 3]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("son"), "ok")

	r = NewRule(`{"substr": ["jsonlogic", 4, -2]}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == D("log"), "ok")

	r = NewRule(`{"log":"apple"}`)
	m, e = r.Apply(nil)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, true, "ok")

	r = NewRule(`{
		"if": [
			{"==": [ { "%": [ { "var": "i" }, 15 ] }, 0]},
			"fizzbuzz",
	
			{"==": [ { "%": [ { "var": "i" }, 3 ] }, 0]},
			"fizz",
	
			{"==": [ { "%": [ { "var": "i" }, 5 ] }, 0]},
			"buzz",
	
			{ "var": "i" }
		]
	}`)
	m, e = r.Apply(`{"i":1}`)
	logger.Debugf("%v %v %v", r.GetExprJSON(), ToJSON(m), e)
	test_utils.AssertTrue(t, true, "ok")

}

func TestCombineRules(t *testing.T) {
	var ruleComponents []map[string]interface{}
	var ruleComponent map[string]interface{}

	person := `{
		"name": {"last": "Simon", "first": "Eric"},
		"age": 40
	}`

	// combine rules (empty rule components)
	rulePointer, err := CombineRules("and", ruleComponents)
	test_utils.AssertTrue(t, err == nil, "Expected CombineRules to succeed")
	test_utils.AssertTrue(t, rulePointer == nil, "Expected nil rule")

	// create first rule
	ruleComponent = R("==", R("var", "name.first"), "Eric")
	ruleComponents = append(ruleComponents, ruleComponent)

	// create second rule
	ruleComponent = R(">", R("var", "age"), 30)
	ruleComponents = append(ruleComponents, ruleComponent)

	// combine rules (and)
	rulePointer, err = CombineRules("and", ruleComponents)
	test_utils.AssertTrue(t, err == nil, "Expected CombineRules to succeed")
	resultMap, err := rulePointer.Apply(person)
	test_utils.AssertTrue(t, err == nil, "Expected Apply to succeed")
	test_utils.AssertTrue(t, resultMap["$result"] == true, "Expected rule to pass")

	// create third rule
	ruleComponent = R("<", R("var", "age"), 35)
	ruleComponents = append(ruleComponents, ruleComponent)

	// combine rules (and)
	rulePointer, err = CombineRules("and", ruleComponents)
	test_utils.AssertTrue(t, err == nil, "Expected CombineRules to succeed")
	resultMap, err = rulePointer.Apply(person)
	test_utils.AssertTrue(t, err == nil, "Expected Apply to succeed")
	test_utils.AssertTrue(t, resultMap["$result"] == false, "Expected rule to fail")

	// combine rules (or)
	rulePointer, err = CombineRules("or", ruleComponents)
	test_utils.AssertTrue(t, err == nil, "Expected CombineRules to succeed")
	resultMap, err = rulePointer.Apply(person)
	test_utils.AssertTrue(t, err == nil, "Expected Apply to succeed")
	test_utils.AssertTrue(t, resultMap["$result"] == true, "Expected rule to pass")

	// combine rules (log)
	rulePointer, err = CombineRules("log", ruleComponents)
	test_utils.AssertTrue(t, err == nil, "Expected CombineRules to succeed")
	resultMap, err = rulePointer.Apply(person)
	test_utils.AssertTrue(t, err == nil, "Expected Apply to succeed")
	expectedResult := `[{"==":[{"var":["name.first"]},"Eric"]},{">":[{"var":["age"]},30]},{"<":[{"var":["age"]},35]}]`
	test_utils.AssertTrue(t, ToJSON(resultMap["$result"]) == expectedResult, "Expected a different result")

	// combine rules (+)
	rulePointer, err = CombineRules("+", ruleComponents)
	test_utils.AssertTrue(t, err == nil, "Expected CombineRules to succeed")
	resultMap, err = rulePointer.Apply(person)
	test_utils.AssertTrue(t, err != nil, "Expected Apply to fail, invalid operator for given rule components")

	// combine rules (empty operator)
	rulePointer, err = CombineRules("", ruleComponents)
	test_utils.AssertTrue(t, err != nil, "Expected CombineRules to fail, empty operator")
}
