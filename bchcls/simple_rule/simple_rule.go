/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package simple_rule is a Go implementation of json-logic represented rules
// that can be used to evaluate data.
package simple_rule

import (
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"

	"encoding/json"
	"math"
	"strconv"
	"strings"
)

var logger = shim.NewLogger("simple_rule_engine")

// Rule represents a rule that can be evaluated over a set of data.
type Rule struct {
	// expr is a rule expression. It has the following format in JSON:
	//
	// { "operator" : [ arg1, arg2, ... ] }
	//
	// The value of args can be any final value (number, string, boolean, array, or map), or a rule expression.
	// Rule expressions are only evaluted when Apply() is called, never during creation.
	// All numeric data in a rule_expression are treated as float64 values.
	// However, if an integer value is required (e.g. % operator), the value is converted to an int64 for internal computation. If the value is not a whole number when an integer is required, an error is returned.
	//
	expr interface{}
	// init is initial data that can be optionally provided. If provided, it will later be merged with the data supplied for actual rule evaluation.
	init map[string]interface{}
}

// NewRule creates a new rule.
// It takes two parameters, rule_expr and init_data (optional), both of which can be either a JSON string or a map[string]interface{}.
// expr is a rule expression. It has the following format in JSON:
//
// { "operator" : [ arg1, arg2, ... ] }
//
// The value of args can be any final value (number, string, boolean, array, or map), or a rule expression.
// Rule expressions are only evaluted when Apply() is called, never during creation.
// All numeric data in a rule_expression are treated as float64 values.
// However, if an integer value is required (e.g. % operator), the value is converted to an int64 for internal computation. If the value is not a whole number when an integer is required, an error is retuned.
//
// init_data is initial data that can be optionally provided. If provided, it will later be merged with the data supplied for actual rule evaluation.
func NewRule(n ...interface{}) Rule {
	r := Rule{}
	var e, i interface{}
	if len(n) == 1 {
		e = n[0]
		i = make(map[string]interface{})
	} else if len(n) == 2 {
		e = n[0]
		i = n[1]
	}
	var expr interface{}
	var init map[string]interface{}

	switch e := e.(type) {
	case string:
		err := json.Unmarshal([]byte(e), &expr)
		if err != nil {
			expr = e
		}
	default:
		expr = e
	}

	switch i := i.(type) {
	case string:
		err := json.Unmarshal([]byte(i), &init)
		if err != nil {
			init = make(map[string]interface{})
		}
	case map[string]interface{}:
		init = i
	default:
		init = make(map[string]interface{})
	}

	r.expr = expr
	r.init = init
	return r
}

// R returns a rule expression given an operator and a set of arguments.
func R(oper string, val ...interface{}) map[string]interface{} {
	var r map[string]interface{}
	r = make(map[string]interface{})
	r[oper] = val
	return r
}

// D returns normalized data as an interface{}.
// If more than one value is supplied, it returns a normalized []interface{}.
func D(n ...interface{}) interface{} {
	if len(n) == 0 {
		return nil
	} else if len(n) == 1 {
		return get64(n[0])
	} else {
		y := []interface{}{}
		for _, x := range n {
			y = append(y, get64(x))
		}
		return y
	}
}

// A returns normalized data as an []interface{}.
func A(n ...interface{}) interface{} {
	if len(n) == 0 {
		return nil
	} else if len(n) == 1 {
		y := []interface{}{get64(n[0])}
		return y
	} else {
		y := []interface{}{}
		for _, x := range n {
			y = append(y, get64(x))
		}
		return y
	}
}

// M returns a map[string]interface{} from a list of values with a []interface{key, value} format.
func M(n ...interface{}) map[string]interface{} {
	val := make(map[string]interface{})
	for _, x := range n {
		switch x := get64(x).(type) {
		case []interface{}:
			if len(x) == 2 {
				switch k := get64(x[0]).(type) {
				case string:
					if len(k) == 0 || k[0:1] == "$" {
						continue
					}
					val[k] = x[1]
				}
			}
		}
	}
	return val
}

// GetExpr returns the rule expression as an interface{}.
func (r *Rule) GetExpr() interface{} {
	return r.expr
}

// GetExprJSON returns the rule expression as a JSON string.
func (r *Rule) GetExprJSON() string {
	return ToJSON(r.expr)
}

// GetInit returns the rule's init data as a map[string]interface{}.
func (r *Rule) GetInit() map[string]interface{} {
	return r.init
}

// GetInitJSON returns the rule's init data as a JSON string.
func (r *Rule) GetInitJSON() string {
	return ToJSON(r.init)
}

// ToJSON returns data as a JSON string.
func ToJSON(e interface{}) string {
	b, err := json.Marshal(&e)
	if err != nil {
		logger.Error(err)
		return ""
	} else {
		s := string(b)
		s = strings.Replace(s, "\\u003e", ">", -1)
		s = strings.Replace(s, "\\u003c", "<", -1)
		s = strings.Replace(s, "\\u0026", "&", -1)
		return s
	}
}

// Apply applies the rule to val and returns the result as a map[string]interface{}.
// val can be a map[string]interface{}, a JSON string, or nil if the data is not needed.
// The evaluted value is stored in the "$result" key of the returned data map.
func (r *Rule) Apply(val interface{}) (map[string]interface{}, error) {
	var return_val map[string]interface{}
	return_val = make(map[string]interface{})
	if r.init != nil {
		for k, v := range r.init {
			return_val[k] = v
		}
	}
	if val != nil {
		val2 := make(map[string]interface{})
		switch val := val.(type) {
		case string:
			ok := json.Unmarshal([]byte(val), &val2)
			if ok != nil {
				val2 = make(map[string]interface{})
				val1 := []interface{}{}
				ok := json.Unmarshal([]byte(val), &val1)
				if ok == nil {
					val2["$current"] = val1
				} else {
					val2["$current"] = val
				}
			}
		case map[string]interface{}:
			val2 = val
		default:
			//try to convert to map[stirng]interface{} first
			val2 = make(map[string]interface{})
			valBytes, err := json.Marshal(&val)
			if valBytes != nil && err == nil {
				ok := json.Unmarshal(valBytes, &val2)
				if ok != nil {
					val2["$current"] = val
				}
			} else {
				val2["$current"] = val
			}
		}

		for k, v := range val2 {
			return_val[k] = v
		}
	}

	result, err := r.eval(r.expr, return_val)
	return_val["$result"] = result
	return return_val, err
}

func (r *Rule) eval(expr interface{}, val map[string]interface{}) (interface{}, error) {
	//logger.Debugf("eval %v %v", expr, val)
	if expr != nil {
		switch x := get64(expr).(type) {
		case int64:
			return float64(x), nil
		case float64:
			return x, nil
		case string:
			return x, nil
		case bool:
			return x, nil
		case []interface{}:
			//return r.evalList(x, val)
			return x, nil
		case map[string]interface{}:
			return r.evalRule(x, val)
		}
	}
	return nil, errors.Errorf("eval: %v invalid type", expr)
}

func (r *Rule) evalList(expr []interface{}, val map[string]interface{}) (interface{}, error) {
	var v []interface{}
	for _, e := range expr {
		x, err := r.eval(e, val)
		if err != nil {
			return nil, err
		} else {
			v = append(v, x)
		}
	}
	return v, nil
}

func (r *Rule) evalMap(expr map[string]interface{}, val map[string]interface{}) (interface{}, error) {
	m := make(map[string]interface{})
	for k, v := range expr {
		x, err := r.eval(v, val)
		if err != nil {
			return nil, err
		} else {
			m[k] = x
		}
	}
	return m, nil
}

func (r *Rule) evalRule(expr map[string]interface{}, val map[string]interface{}) (interface{}, error) {

	if (len(expr)) == 1 {
		for k, v := range expr {
			switch v := v.(type) {
			case []interface{}:
				switch k {
				case "log":
					return r.log(v, val)
				case "+":
					return r.add(v, val)
				case "-":
					return r.sub(v, val)
				case "*":
					return r.mult(v, val)
				case "/":
					return r.div(v, val)
				case "%":
					return r.mod(v, val)
				case "max":
					return r.max(v, val)
				case "min":
					return r.min(v, val)
				case "int":
					return r.integer(v, val)
				case "float":
					return r.float(v, val)
				case "round":
					return r.round(v, val)
				case "floor":
					return r.floor(v, val)
				case "ceil":
					return r.ceil(v, val)
				case "sqrt":
					return r.sqrt(v, val)
				case "cat":
					return r.cat(v, val)
				case "contains":
					return r.contains(v, val)
				case "substr":
					return r.substr(v, val)
				case "===":
					return r.eq_strict(v, val)
				case "==":
					return r.eq(v, val)
				case "!=":
					return r.neq(v, val)
				case "!==":
					return r.neq_strict(v, val)
				case ">":
					return r.gt(v, val)
				case ">=":
					return r.gte(v, val)
				case "<":
					return r.lt(v, val)
				case "<=":
					return r.lte(v, val)
				case "!":
					return r.not(v, val)
				case "!!":
					return r.boolean(v, val)
				case "bool":
					return r.boolean(v, val)
				case "not":
					return r.not(v, val)
				case "and":
					return r.and(v, val)
				case "&&":
					return r.and(v, val)
				case "or":
					return r.or(v, val)
				case "||":
					return r.or(v, val)
				case "if":
					return r.cond(v, val)
				case "var":
					return r.variable(v, val)
				case "missing":
					return r.missing(v, val)
				case "missing_some":
					return r.missing_some(v, val)
				case "let":
					return r.let(v, val)
				case ":=":
					return r.let(v, val)
				case "array":
					return r.array(v, val)
				case "list":
					return r.array(v, val)
				case "in":
					return r.in(v, val)
				case "map":
					return r.map_array(v, val)
				case "for_each":
					return r.map_array(v, val)
				case "filter":
					return r.filter_array(v, val)
				case "reduce":
					return r.reduce_array(v, val)
				case "all":
					return r.all_array(v, val)
				case "some":
					return r.some_array(v, val)
				case "none":
					return r.none_array(v, val)
				case "merge":
					return r.merge_array(v, val)
				case "len":
					return r.len_array(v, val)
				case "dict":
					return r.dict(v, val)
				case "keys":
					return r.keys_map(v, val)
				case "has_key":
					return r.has_key_map(v, val)
				case "get":
					return r.get(v, val)
				case "proc":
					return r.proc(v, val)
				case "while":
					return r.while_proc(v, val)
				}
			//support single value for following operators
			case interface{}:

				/*
					switch k {
					case "int":
						return r.integer(v, val)
					case "float":
						return r.float(v, val)
					case "round":
						return r.round(v, val)
					case "floor":
						return r.floor(v, val)
					case "ceil":
						return r.floor(v, val)
					case "sqrt":
						return r.floor(v, val)
					case "!":
						return r.not(v, val)
					case "!!":
						return r.boolean(v, val)
					case "bool":
						return r.boolean(v, val)
					case "not":
						return r.not(v, val)

					case "-":
						return r.sub([]interface{}{v}, val)
					case "var":
						return r.variable([]interface{}{v}, val)
					case "keys":
						return r.keys_map([]interface{}{v}, val)
					case "+":
						return r.float([]interface{}{v}, val)
					}
				*/

				x, e := r.eval(v, val)
				if e == nil {
					if v, ok := x.([]interface{}); ok {
						switch k {
						case "log":
							return r.log(v, val)
						case "+":
							return r.add(v, val)
						case "-":
							return r.sub(v, val)
						case "*":
							return r.mult(v, val)
						case "/":
							return r.div(v, val)
						case "%":
							return r.mod(v, val)
						case "max":
							return r.max(v, val)
						case "min":
							return r.min(v, val)
						case "int":
							return r.integer(v, val)
						case "float":
							return r.float(v, val)
						case "round":
							return r.round(v, val)
						case "floor":
							return r.floor(v, val)
						case "ceil":
							return r.ceil(v, val)
						case "sqrt":
							return r.sqrt(v, val)
						case "cat":
							return r.cat(v, val)
						case "contains":
							return r.contains(v, val)
						case "substr":
							return r.substr(v, val)
						case "===":
							return r.eq_strict(v, val)
						case "==":
							return r.eq(v, val)
						case "!=":
							return r.neq(v, val)
						case "!==":
							return r.neq_strict(v, val)
						case ">":
							return r.gt(v, val)
						case ">=":
							return r.gte(v, val)
						case "<":
							return r.lt(v, val)
						case "<=":
							return r.lte(v, val)
						case "!":
							return r.not(v, val)
						case "!!":
							return r.boolean(v, val)
						case "bool":
							return r.boolean(v, val)
						case "not":
							return r.not(v, val)
						case "and":
							return r.and(v, val)
						case "&&":
							return r.and(v, val)
						case "or":
							return r.or(v, val)
						case "||":
							return r.or(v, val)
						case "if":
							return r.cond(v, val)
						case "var":
							return r.variable(v, val)
						case "missing":
							return r.missing(v, val)
						case "missing_some":
							return r.missing_some(v, val)
						case "let":
							return r.let(v, val)
						case ":=":
							return r.let(v, val)
						case "array":
							return r.array(v, val)
						case "list":
							return r.array(v, val)
						case "in":
							return r.in(v, val)
						case "map":
							return r.map_array(v, val)
						case "for_each":
							return r.map_array(v, val)
						case "filter":
							return r.filter_array(v, val)
						case "reduce":
							return r.reduce_array(v, val)
						case "all":
							return r.all_array(v, val)
						case "some":
							return r.some_array(v, val)
						case "none":
							return r.none_array(v, val)
						case "merge":
							return r.merge_array(v, val)
						case "len":
							return r.len_array(v, val)
						case "dict":
							return r.dict(v, val)
						case "keys":
							return r.keys_map(v, val)
						case "has_key":
							return r.has_key_map(v, val)
						case "get":
							return r.get(v, val)
						case "proc":
							return r.proc(v, val)
						case "while":
							return r.while_proc(v, val)
						}
					}
				}

				switch k {
				case "log":
					return r.log([]interface{}{v}, val)
				case "+":
					return r.add([]interface{}{v}, val)
				case "-":
					return r.sub([]interface{}{v}, val)
				case "*":
					return r.mult([]interface{}{v}, val)
				case "/":
					return r.div([]interface{}{v}, val)
				case "%":
					return r.mod([]interface{}{v}, val)
				case "max":
					return r.max([]interface{}{v}, val)
				case "min":
					return r.min([]interface{}{v}, val)
				case "int":
					return r.integer([]interface{}{v}, val)
				case "float":
					return r.float([]interface{}{v}, val)
				case "round":
					return r.round([]interface{}{v}, val)
				case "floor":
					return r.floor([]interface{}{v}, val)
				case "ceil":
					return r.ceil([]interface{}{v}, val)
				case "sqrt":
					return r.sqrt([]interface{}{v}, val)
				case "cat":
					return r.cat([]interface{}{v}, val)
				case "contains":
					return r.contains([]interface{}{v}, val)
				case "substr":
					return r.substr([]interface{}{v}, val)
				case "===":
					return r.eq_strict([]interface{}{v}, val)
				case "==":
					return r.eq([]interface{}{v}, val)
				case "!=":
					return r.neq([]interface{}{v}, val)
				case "!==":
					return r.neq_strict([]interface{}{v}, val)
				case ">":
					return r.gt([]interface{}{v}, val)
				case ">=":
					return r.gte([]interface{}{v}, val)
				case "<":
					return r.lt([]interface{}{v}, val)
				case "<=":
					return r.lte([]interface{}{v}, val)
				case "!":
					return r.not([]interface{}{v}, val)
				case "!!":
					return r.boolean([]interface{}{v}, val)
				case "bool":
					return r.boolean([]interface{}{v}, val)
				case "not":
					return r.not([]interface{}{v}, val)
				case "and":
					return r.and([]interface{}{v}, val)
				case "&&":
					return r.and([]interface{}{v}, val)
				case "or":
					return r.or([]interface{}{v}, val)
				case "||":
					return r.or([]interface{}{v}, val)
				case "if":
					return r.cond([]interface{}{v}, val)
				case "var":
					return r.variable([]interface{}{v}, val)
				case "missing":
					return r.missing([]interface{}{v}, val)
				case "missing_some":
					return r.missing_some([]interface{}{v}, val)
				case "let":
					return r.let([]interface{}{v}, val)
				case ":=":
					return r.let([]interface{}{v}, val)
				case "array":
					return r.array([]interface{}{v}, val)
				case "list":
					return r.array([]interface{}{v}, val)
				case "in":
					return r.in([]interface{}{v}, val)
				case "map":
					return r.map_array([]interface{}{v}, val)
				case "for_each":
					return r.map_array([]interface{}{v}, val)
				case "filter":
					return r.filter_array([]interface{}{v}, val)
				case "reduce":
					return r.reduce_array([]interface{}{v}, val)
				case "all":
					return r.all_array([]interface{}{v}, val)
				case "some":
					return r.some_array([]interface{}{v}, val)
				case "none":
					return r.none_array([]interface{}{v}, val)
				case "merge":
					return r.merge_array([]interface{}{v}, val)
				case "len":
					return r.len_array([]interface{}{v}, val)
				case "dict":
					return r.dict([]interface{}{v}, val)
				case "keys":
					return r.keys_map([]interface{}{v}, val)
				case "has_key":
					return r.has_key_map([]interface{}{v}, val)
				case "get":
					return r.get([]interface{}{v}, val)
				case "proc":
					return r.proc([]interface{}{v}, val)
				case "while":
					return r.while_proc([]interface{}{v}, val)
				}

			}
		}
	}
	//return r.evalMap(expr, val)
	return expr, nil
}

//converts to all number and list
func get64(x interface{}) interface{} {
	switch x := x.(type) {
	case uint8:
		return float64(x)
	case int8:
		return float64(x)
	case uint16:
		return float64(x)
	case int16:
		return float64(x)
	case uint32:
		return float64(x)
	case int32:
		return float64(x)
	case uint64:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case int:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return float64(x)
	case []uint8:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []int8:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []uint16:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []int16:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []uint32:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []int32:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []uint64:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []int64:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []uint:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []int:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []float32:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []float64:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []string:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	case []bool:
		var x2 []interface{}
		x3, _ := json.Marshal(&x)
		json.Unmarshal(x3, &x2)
		return x2
	}
	return x
}

// Math ********************

func (r *Rule) add(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("add: no args")
	}

	var y float64 = 0
	for _, x := range n {

		x, e := r.float(x, v)
		if e != nil {
			return y, errors.Errorf("add: %e", e)
		}

		y = y + x.(float64)
	}

	return y, nil

}

func (r *Rule) sub(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("sub: no args")
	}

	var y float64 = 0
	x, e := r.float(n[0], v)
	if e != nil {
		return nil, errors.Errorf("sub: invalid input %v", e)
	}
	y = x.(float64)
	if len(n) == 1 {
		return y * -1, nil
	}

	for _, x := range n[1:] {

		x, e := r.float(x, v)
		if e != nil {
			return y, errors.Errorf("sub: %e", e)
		}

		y = y - x.(float64)
	}

	return y, nil
}

func (r *Rule) mult(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("mult: no args")
	}

	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("mult: %v", e)
	}

	switch x := x.(type) {
	case int64:
		if len(n) == 1 {
			return float64(x), nil
		}
		if x == 0 {
			return float64(0), nil
		}
		y, e := r.mult(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case int64:
			return float64(x * y), nil
		case float64:
			return float64(x) * y, nil
		}
		return nil, errors.Errorf("mult: %v is not numeric type", y)
	case float64:
		if len(n) == 1 {
			return x, nil
		}
		if x == 0 {
			return float64(0), nil
		}
		y, e := r.mult(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case int64:
			return x * float64(y), nil
		case float64:
			return x * y, nil
		}
		return nil, errors.Errorf("mult: %v is not numeric type", y)
	}
	return nil, errors.Errorf("mult: %v is not numeric type", x)
}

func (r *Rule) div(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("div: no args")
	}

	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("div: %v", e)
	}

	switch x := x.(type) {
	case int64:
		if len(n) == 1 {
			return float64(x), nil
		}

		var z float64
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("div: %v", e)
		}
		switch y := y.(type) {
		case int64:
			if y == 0 {
				return nil, errors.New("div: div by 0")
			}
			z = float64(x) / float64(y)
		case float64:
			if y == 0 {
				return nil, errors.New("div: div by 0")
			}
			z = float64(x) / y
		default:
			return nil, errors.Errorf("div: %v is not numeric type", y)
		}

		if len(n) == 2 {
			return z, nil
		}
		return r.div(append([]interface{}{z}, n[2:]...), v)
	case float64:
		if len(n) == 1 {
			return x, nil
		}

		var z float64
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("div: %v", e)
		}
		switch y := y.(type) {
		case int64:
			if y == 0 {
				return nil, errors.New("div: div by 0")
			}
			z = x / float64(y)
		case float64:
			if y == 0 {
				return nil, errors.New("div: div by 0")
			}
			z = x / y
		default:
			return nil, errors.Errorf("div: %v is not numeric type", y)
		}

		if len(n) == 2 {
			return z, nil
		}
		return r.div(append([]interface{}{z}, n[2:]...), v)
	}
	return nil, errors.Errorf("div: %v is not numeric type", x)
}

func (r *Rule) mod(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("mod: no args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("mod: %v", e)
	}

	switch x := x.(type) {
	case int64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("mod: %v", e)
		}
		switch y := y.(type) {
		case int64:
			return float64(x % y), nil
		case float64:
			if y == float64(int64(y)) {
				return float64(x % int64(y)), nil
			} else {
				return nil, errors.Errorf("mod: %v is not int value", y)
			}
		}
		return nil, errors.Errorf("mod: %v is not numeric type", y)
	case float64:
		if x == float64(int64(x)) {
			y, e := r.eval(n[1], v)
			if e != nil {
				return nil, errors.Errorf("mod: %v", e)
			}
			switch y := y.(type) {
			case int64:
				return float64(int64(x) % y), nil
			case float64:
				if y == float64(int64(y)) {
					return float64(int64(x) % int64(y)), nil
				} else {
					return nil, errors.Errorf("mod: %v is not int value", y)
				}
			}
			return nil, errors.Errorf("mod: %v is not numeric type", y)
		} else {
			return nil, errors.Errorf("mod: %v is not int value", x)
		}
	}
	return nil, errors.Errorf("mod: %v is not numeric type", x)
}

func (r *Rule) max(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("max: no args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("max: %v", e)
	}
	switch x := x.(type) {
	case int64:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.max(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case int64:
			if x < y {
				return float64(y), nil
			} else {
				return float64(x), nil
			}
		case float64:
			if float64(x) < y {
				return y, nil
			} else {
				return float64(x), nil
			}
		}
		return nil, errors.Errorf("max: %v %v type mismatch", x, y)
	case float64:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.max(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case int64:
			if x < float64(y) {
				return float64(y), nil
			} else {
				return x, nil
			}
		case float64:
			if x < y {
				return y, nil
			} else {
				return x, nil
			}
		}
		return nil, errors.Errorf("max: %v %v type mismatch", x, y)
	case string:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.max(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case string:
			if x < y {
				return y, nil
			} else {
				return x, nil
			}
		}
		return nil, errors.Errorf("max: %v %v type mismatch", x, y)
	}
	return nil, errors.Errorf("max: %v is not a suppoted type", x)
}

func (r *Rule) min(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("min: no args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("min: %v", e)
	}
	switch x := x.(type) {
	case int64:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.min(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case int64:
			if x > y {
				return float64(y), nil
			} else {
				return float64(x), nil
			}
		case float64:
			if float64(x) > y {
				return y, nil
			} else {
				return float64(x), nil
			}
		}
		return nil, errors.Errorf("min: %v %v type mismatch", x, y)
	case float64:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.min(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case int64:
			if x > float64(y) {
				return float64(y), nil
			} else {
				return x, nil
			}
		case float64:
			if x > y {
				return y, nil
			} else {
				return x, nil
			}
		}
		return nil, errors.Errorf("max: %v %v type mismatch", x, y)
	case string:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.min(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case string:
			if x > y {
				return y, nil
			} else {
				return x, nil
			}
		}
		return nil, errors.Errorf("max: %v %v type mismatch", x, y)
	}

	return nil, errors.Errorf("min: %v is not a suppoted type", x)
}

func (r *Rule) boolean(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("bool: %v", e)
	}
	if x == nil {
		return false, nil
	}
	switch x := x.(type) {
	case bool:
		return x, nil
	case int64:
		return x != 0, nil
	case float64:
		return x != 0, nil
	case string:
		return len(x) > 0, nil
	case []interface{}:
		if len(x) == 0 {
			return false, nil
		} else if len(x) > 1 {
			return true, nil
		} else {
			return r.boolean(x[0], v)
		}
	}
	return true, nil
}

func (r *Rule) boolean_eval(n interface{}, v map[string]interface{}) bool {
	x, e := r.eval(n, v)
	if e != nil {
		return false
	}
	if x == nil {
		return false
	}
	switch x := x.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return len(x) > 0
	case []interface{}:
		return len(x) == 0
	}
	return true
}

func (r *Rule) integer(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("int: %v", e)
	}
	switch x := x.(type) {
	case int64:
		return float64(x), nil
	case float64:
		return float64(int64(x)), nil
	case string:
		//y, err := strconv.ParseInt(x, 10, 64)
		y, err := strconv.ParseFloat(x, 64)
		if err == nil {
			return float64(int64(y)), nil
		} else {
			return nil, errors.Errorf("int: %v", err)
		}
	case []interface{}:
		if len(x) == 1 {
			return r.integer(x[0], v)
		} else {
			return nil, errors.New("int: wrong number of args")
		}
	}
	return nil, errors.Errorf("int: %v invalid type", x)
}

func (r *Rule) float(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("float: %v", e)
	}
	switch x := x.(type) {
	case int64:
		return float64(x), nil
	case float64:
		return x, nil
	case string:
		y, err := strconv.ParseFloat(x, 64)
		if err == nil {
			return y, nil
		} else {
			return nil, errors.Errorf("float: %v", err)
		}
	case []interface{}:
		if len(x) == 1 {
			return r.float(x[0], v)
		} else {
			return nil, errors.New("float: wrong number of args")
		}
	}
	return nil, errors.Errorf("float: %v invalid type", x)
}

func (r *Rule) round(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("round: %v", e)
	}
	switch x := x.(type) {
	case int64:
		return float64(x), nil
	case float64:
		if x > 0 {
			return float64(int64(math.Floor(x + 0.5))), nil
		} else {
			return float64(int64(math.Ceil(x - 0.5))), nil
		}

	case []interface{}:
		if len(x) == 1 {
			return r.round(x[0], v)
		} else {
			return nil, errors.New("round: wrong number of args")
		}
	}
	return nil, errors.Errorf("round: %v invalid type", x)
}

func (r *Rule) floor(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("floor: %v", e)
	}
	switch x := x.(type) {
	case int64:
		return float64(x), nil
	case float64:
		return float64(int64(math.Floor(x))), nil
	case []interface{}:
		if len(x) == 1 {
			return r.floor(x[0], v)
		} else {
			return nil, errors.New("floor: wrong number of args")
		}
	}
	return nil, errors.Errorf("floor: %v invalid type", x)
}

func (r *Rule) ceil(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("ceil: %v", e)
	}
	switch x := x.(type) {
	case int64:
		return float64(x), nil
	case float64:
		return float64(int64(math.Ceil(x))), nil
	case []interface{}:
		if len(x) == 1 {
			return r.ceil(x[0], v)
		} else {
			return nil, errors.New("ceil: wrong number of args")
		}
	}
	return nil, errors.Errorf("ceil: %v invalid type", x)
}

func (r *Rule) sqrt(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("sqrt: %v", e)
	}
	switch x := x.(type) {
	case int64:
		if x < 0 {
			return nil, errors.Errorf("sqrt: %v negative squrt undefined", x)
		} else {
			return math.Sqrt(float64(x)), nil
		}
	case float64:
		if x < 0 {
			return nil, errors.Errorf("sqrt: %v negative squrt undefined", x)
		} else {
			return math.Sqrt(x), nil
		}

	case []interface{}:
		if len(x) == 1 {
			return r.sqrt(x[0], v)
		} else {
			return nil, errors.New("sqrt: wrong number of args")
		}
	}
	return nil, errors.Errorf("sqrt: %v invalid type", x)
}

// String ********************

func (r *Rule) cat(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("cat: no args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("cat: %v", e)
	}
	switch x := x.(type) {
	case string:
		if len(n) == 1 {
			return x, nil
		}
		y, e := r.cat(n[1:], v)
		if e != nil {
			return nil, e
		}
		switch y := y.(type) {
		case string:
			return x + y, nil
		}
		return nil, errors.Errorf("cat: %v is not string type", y)
	}
	return nil, errors.Errorf("cat: %v is not string type", x)
}

func (r *Rule) contains(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("contains: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("contains: %v", e)
	}
	switch x := x.(type) {
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("contains: %v", e)
		}
		switch y := y.(type) {
		case string:
			return strings.Contains(x, y), nil
		}
		return nil, errors.Errorf("contains: %v is not string type", y)
	}
	return nil, errors.Errorf("contains; %v is not string type", x)
}

func (r *Rule) substr(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 || len(n) > 3 {
		return nil, errors.New("substr: worng number of args")
	}
	s, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("substr: %v", e)
	}
	if s, ok := s.(string); ok {
		if len(n) == 1 {
			return s, nil
		}
		sn := int64(len(s))

		bi, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("substr: %v", e)
		}
		switch b2 := bi.(type) {
		case float64:
			if b2 == float64(int64(b2)) {
				bi = int64(b2)
			}
		}
		if bi, ok := bi.(int64); ok {
			if bi < 0 {
				bi = sn + bi
			}
			ei := sn
			if len(n) == 3 {
				c, e := r.eval(n[2], v)
				if e != nil {
					return nil, errors.Errorf("substr: %v", e)
				}
				switch c2 := c.(type) {
				case float64:
					if c2 == float64(int64(c2)) {
						c = int64(c2)
					}
				}
				if c, ok := c.(int64); ok {
					if c < 0 {
						ei = sn + c
					} else {
						ei = bi + c
						if ei > sn {
							ei = sn
						}
					}
				} else {
					return nil, errors.Errorf("substr: %v is not numeric type", c)
				}
			}

			if bi < 0 || bi > sn {
				return nil, errors.New("subster: index out of range")
			}
			if ei < bi || ei > sn {
				return nil, errors.New("substr: index out ouf range")
			}
			return s[bi:ei], nil
		}
	}
	return nil, errors.Errorf("substr: %v is not string type", s)
}

// Compare ********************

func (r *Rule) eq(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("eq: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("eq: %v", e)
	}
	switch x := x.(type) {
	case float64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq: %v", e)
		}
		switch y := y.(type) {
		case float64:
			return x == y, nil
		case string:
			y1, e := r.float(y, v)
			if e == nil {
				return x == y1.(float64), nil
			}
		case bool:
			x1, e := r.boolean(x, v)
			if e == nil {
				return x1.(bool) == y, nil
			}
		}
		return false, nil
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq: %v", e)
		}
		switch y := y.(type) {
		case string:
			return x == y, nil
		case float64:
			x1, e := r.float(x, v)
			if e == nil {
				return x1.(float64) == y, nil
			}
		case bool:
			x1, e := r.boolean(x, v)
			if e == nil {
				return x1.(bool) == y, nil
			}
		}
		return false, nil
	case bool:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq: %v", e)
		}
		switch y := y.(type) {
		case bool:
			return x == y, nil
		default:
			y1, e := r.boolean(y, v)
			if e == nil {
				return x == y1.(bool), nil
			}
		}
		return false, nil
	default:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq: %v", e)
		}
		switch y := y.(type) {
		case bool:
			x1, e := r.boolean(x, v)
			if e == nil {
				return x1.(bool) == y, nil
			}
		default:
			return x == y, nil
		}
	}
	return false, nil
}

func (r *Rule) eq_strict(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("eq strict: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("eq strict: %v", e)
	}
	switch x := x.(type) {
	case int64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq strict: %v", e)
		}
		switch y := y.(type) {
		case int64:
			return x == y, nil
		case float64:
			return float64(x) == y, nil
		}
		return false, nil
	case float64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq strict: %v", e)
		}
		switch y := y.(type) {
		case int64:
			return x == float64(y), nil
		case float64:
			return x == y, nil
		}
		return false, nil
	default:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("eq strict: %v", e)
		}
		switch y := y.(type) {
		default:
			return x == y, nil
		}
	}
	return false, nil
}

func (r *Rule) neq(n []interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eq(n, v)
	if e != nil {
		return nil, errors.Errorf("neq: %v", e)
	}

	return !x.(bool), nil
}

func (r *Rule) neq_strict(n []interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eq_strict(n, v)
	if e != nil {
		return nil, errors.Errorf("neq strict: %v", e)
	}

	return !x.(bool), nil
}

func (r *Rule) gt(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 && len(n) != 3 {
		return nil, errors.New("gt: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("gt: %v", e)
	}
	switch x := x.(type) {
	case float64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("gt: %v", e)
		}
		switch y := y.(type) {
		case float64:
			if len(n) == 2 {
				return x > y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("gt: %v", e)
			}
			switch z := z.(type) {
			case float64:
				return x > y && y > z, nil
			}
		}
		return nil, errors.New("gt: type mismatch")
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("gt: %v", e)
		}
		switch y := y.(type) {
		case string:
			if len(n) == 2 {
				return x > y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("gt: %v", e)
			}
			switch z := z.(type) {
			case string:
				return x > y && y > z, nil
			}
		}
		return nil, errors.New("gt: type mismatch")
	}
	return nil, errors.New("gt: unsupported type")
}

func (r *Rule) gte(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 && len(n) != 3 {
		return nil, errors.New("gte: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("gte: %v", e)
	}
	switch x := x.(type) {
	case float64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("gte: %v", e)
		}
		switch y := y.(type) {
		case float64:
			if len(n) == 2 {
				return x >= y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("gte: %v", e)
			}
			switch z := z.(type) {
			case float64:
				return x >= y && y >= z, nil
			}
		}
		return nil, errors.New("gte: type mismatch")
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("gte: %v", e)
		}
		switch y := y.(type) {
		case string:
			if len(n) == 2 {
				return x >= y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("gte: %v", e)
			}
			switch z := z.(type) {
			case string:
				return x >= y && y >= z, nil
			}
		}
		return nil, errors.New("gte: type mismatch")
	}
	return nil, errors.New("gte: unsupported type")
}

func (r *Rule) lt(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 && len(n) != 3 {
		return nil, errors.New("lt: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("lt: %v", e)
	}
	switch x := x.(type) {
	case float64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("lt: %v", e)
		}
		switch y := y.(type) {
		case float64:
			if len(n) == 2 {
				return x < y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("lt: %v", e)
			}
			switch z := z.(type) {
			case float64:
				return x < y && y < z, nil
			}
		}
		return nil, errors.New("lt: type mismatch")
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("lt: %v", e)
		}
		switch y := y.(type) {
		case string:
			if len(n) == 2 {
				return x < y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("lt: %v", e)
			}
			switch z := z.(type) {
			case string:
				return x < y && y < z, nil
			}
		}
		return nil, errors.New("lt: type mismatch")
	}
	return nil, errors.New("lt: unsupported type")
}

func (r *Rule) lte(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 && len(n) != 3 {
		return nil, errors.New("lte: wrong number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("lte: %v", e)
	}
	switch x := x.(type) {
	case float64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("lte: %v", e)
		}
		switch y := y.(type) {
		case float64:
			if len(n) == 2 {
				return x <= y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("lte: %v", e)
			}
			switch z := z.(type) {
			case float64:
				return x <= y && y <= z, nil
			}
		}
		return nil, errors.New("lte: type mismatch")
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("lte: %v", e)
		}
		switch y := y.(type) {
		case string:
			if len(n) == 2 {
				return x <= y, nil
			}
			z, e := r.eval(n[2], v)
			if e != nil {
				return nil, errors.Errorf("lte: %v", e)
			}
			switch z := z.(type) {
			case string:
				return x <= y && y <= z, nil
			}
		}
		return nil, errors.New("lte: type mismatch")
	}
	return nil, errors.New("lte: unsupported type")
}

// Bool ********************

func (r *Rule) not(n interface{}, v map[string]interface{}) (interface{}, error) {
	x, e := r.eval(n, v)
	if e != nil {
		return nil, errors.Errorf("not: %v", e)
	}
	switch x := x.(type) {
	case bool:
		return !x, nil
	case []interface{}:
		if len(x) == 1 {
			return r.not(x[0], v)
		}
	}
	return nil, errors.Errorf("not: %v is not bool type", x)
}

func (r *Rule) and(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("and: no args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("and: %v", e)
	}
	if len(n) == 1 {
		return x, nil
	}

	x0 := r.boolean_eval(x, v)

	if x0 == false {
		return x, nil
	}

	return r.and(n[1:], v)
}

func (r *Rule) or(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("or: no args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("or: %v", e)
	}
	if len(n) == 1 {
		return x, nil
	}

	x0 := r.boolean_eval(x, v)

	if x0 == true {
		return x, nil
	}

	return r.or(n[1:], v)
}

//[if, then, else]
//[if, then, esle if, then, else if, then ]
func (r *Rule) cond(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) < 2 {
		return nil, errors.New("if: wrong number of args")
	}
	c, e := r.boolean(n[0], v)
	if e != nil {
		return nil, errors.Errorf("if: %v", e)
	}

	if c, ok := c.(bool); ok {
		if len(n) <= 3 {
			if c {
				return r.eval(n[1], v)
			} else if len(n) == 3 {
				return r.eval(n[2], v)
			} else {
				return nil, nil
			}
		} else {
			if c {
				return r.eval(n[1], v)
			} else {
				return r.cond(n[2:], v)
			}
		}
	}

	return nil, errors.Errorf("if: %v is not bool type", c)

}

// Var ********************

func (r *Rule) variable_eval(val interface{}, n []string, d interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return val, nil
	}

	switch val := val.(type) {
	case map[string]interface{}:

		if v2, ok := val[n[0]]; ok {
			v3, e := r.eval(v2, nil)
			if e != nil {
				return nil, errors.Errorf("var: %v", e)
			}
			if len(n) == 1 {
				return v3, nil
			} else {
				return r.variable_eval(v3, n[1:], d, v)
			}
		} else {
			if d != nil {
				x, e := r.eval(d, v)
				if e != nil {
					return nil, errors.Errorf("var: %v", e)
				}
				return x, nil
			} else {
				return nil, nil
			}
		}

	case []interface{}:
		x, e := r.integer(n[0], v)
		if e != nil {
			return nil, errors.Errorf("var: invalid index: %v", e)
		}
		if k, ok := x.(float64); ok {
			if int64(len(val)) > int64(k) && int64(k) >= 0 {
				v2, e := r.eval(val[int64(k)], nil)
				if e != nil {
					return nil, errors.Errorf("var: %v", e)
				}
				if len(n) == 1 {
					return v2, nil
				} else {
					return r.variable_eval(v2, n[1:], d, v)
				}
			} else {
				return nil, errors.Errorf("var: index out of range: %v", k)
			}
		}

		return nil, errors.Errorf("var: invalid index: %v", x)
	}
	return nil, errors.Errorf("var: data is not a map or array type: %v", nil)
}

func (r *Rule) variable(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 || len(n) > 2 {
		return nil, errors.New("var: invalid number of args")
	}

	var d interface{} = nil
	if len(n) == 2 {
		d = n[1]
	}
	vname, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("var: invalid variable name: %v", e)
	}

	if index, ok := vname.(float64); ok {
		if index == float64(int64(index)) {
			vname = "." + strconv.Itoa(int(index))
		}
	}
	if vname, ok := vname.(string); ok {

		vname2 := strings.Split(vname, ".")

		if len(vname2[0]) == 0 {
			vname2[0] = "$current"
		}
		if vname2[0] == "current" {
			vname2[0] = "$current"
		}
		if vname2[0] == "accumulator" {
			vname2[0] = "$accumulator"
		}
		val, ok := v[vname2[0]]
		if !ok {
			if d != nil {
				x, e := r.eval(d, v)
				if e != nil {
					return nil, errors.Errorf("var: %v", e)
				}
				return x, nil
			} else {
				return nil, nil
				//return nil, errors.Errorf("var: %v does not exist in the data map", vname)
			}
		}

		x, e := r.eval(val, nil)
		if e != nil {
			return nil, errors.Errorf("var: invalid data value: %v", e)
		}
		if len(vname2) == 1 {
			return x, nil
		}

		return r.variable_eval(x, vname2[1:], d, v)

	}
	return nil, errors.Errorf("var: %v invalid variable name", n[0])
}

func (r *Rule) missing(n []interface{}, v map[string]interface{}) (interface{}, error) {
	y := []interface{}{}
	if len(n) == 0 {
		return y, nil
	}
	for _, k := range n {
		x, e := r.eval(k, v)
		if e != nil {
			return nil, errors.Errorf("missing: invalid input: %v", e)
		}
		switch x := x.(type) {
		case string:
			if _, ok := v[x]; !ok {
				y = append(y, x)
			}
		}
	}
	return y, nil
}

func (r *Rule) missing_some(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return []interface{}{}, errors.New("missing_some: wrong number of args")
	}

	num, e := r.eval(n[0], v)
	if e != nil {
		return []interface{}{}, errors.Errorf("missing_some: invalid minimum %v", e)
	}
	if num, ok := num.(float64); ok {
		if num == float64(int64(num)) {
			n1, e := r.eval(n[1], v)
			if e != nil {
				return []interface{}{}, errors.Errorf("missing_some: invalide arg %v", e)
			}
			if n1, ok := n1.([]interface{}); ok {
				x, e := r.missing(n1, v)
				if e != nil {
					return []interface{}{}, errors.Errorf("missing_some: %v", e)
				}

				count := 0
				if x, ok := x.([]interface{}); ok {
					count = len(x)
				}

				if int64(len(n1)-count) >= int64(num) {
					return []interface{}{}, nil
				} else {
					return x, nil
				}
			} else {
				return []interface{}{}, errors.Errorf("missing_some: not an array %v", n1)
			}
		} else {
			return []interface{}{}, errors.Errorf("missing_some: invalid minimun %v", num)
		}
	} else {
		return []interface{}{}, errors.New("missing_some: invalid minimun arg")
	}
}

func (r *Rule) let(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 2 {
		if vname, ok := get64(n[0]).(string); ok {
			if len(vname) == 0 || vname[0:1] == "$" {
				return nil, errors.Errorf("let: %v invalid variable name", vname)
			}
			val, e := r.eval(n[1], v)
			if e != nil {
				return nil, errors.Errorf("let: %v", e)
			}
			v[vname] = val
			return val, nil
		}
	}
	// array or map
	if len(n) == 3 {
		if vname, ok := get64(n[0]).(string); ok {
			if len(vname) == 0 || vname[0:1] == "$" {
				return nil, errors.Errorf("let: %v invalid variable name", vname)
			}
			val, ok := v[vname]
			if !ok {
				return nil, errors.Errorf("let: %v undefined variable", vname)
			}
			x, e := r.eval(val, nil)
			if e != nil {
				return nil, errors.Errorf("let: invalid data value: %v", e)
			}
			switch val := x.(type) {
			case map[string]interface{}:
				vname2, e := r.eval(n[1], v)
				if e != nil {
					return nil, errors.Errorf("let: invalid key: %v", e)
				}
				switch vname2 := vname2.(type) {
				case string:
					val2, e := r.eval(n[2], v)
					if e != nil {
						return nil, errors.Errorf("let: %v", e)
					}
					val[vname2] = val2
					v[vname] = val
					return val2, nil
				}
				return nil, errors.Errorf("let: %v invalid key", vname2)
			case []interface{}:
				i, e := r.eval(n[1], v)
				if e != nil {
					return nil, errors.Errorf("let: invalid index: %v", e)
				}
				switch i := i.(type) {
				case int64:
					if i < int64(len(val)) {
						val2, e := r.eval(n[2], v)
						if e != nil {
							return nil, errors.Errorf("let: %v", e)
						}
						if i >= 0 {
							val[i] = val2
						} else {
							val = append(val, val2)
						}
						v[vname] = val
						return val2, nil
					}
					return nil, errors.Errorf("let: %v invalid index for %v", i, vname)
				case float64:
					if i == float64(int64(i)) {
						if int64(i) < int64(len(val)) {
							val2, e := r.eval(n[2], v)
							if e != nil {
								return nil, errors.Errorf("let: %v", e)
							}
							if int64(i) >= 0 {
								val[int64(i)] = val2
							} else {
								val = append(val, val2)
							}
							v[vname] = val
							return val2, nil
						}
					}
					return nil, errors.Errorf("let: %v invalid index for %v", i, vname)
				}
			}
		}
	}
	return nil, errors.New("let: invalid number of args")
}

// Array ********************

func (r *Rule) dict(n []interface{}, v map[string]interface{}) (interface{}, error) {
	val := make(map[string]interface{})
	for _, x := range n {
		x, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("dict: %v", e)
		}
		switch x := x.(type) {
		case []interface{}:
			if len(x) == 2 {
				k, e := r.eval(x[0], v)
				if e != nil {
					return nil, errors.Errorf("dict: invalid key %v", e)
				}
				switch k := k.(type) {
				case string:
					if len(k) == 0 || k[0:1] == "$" {
						return nil, errors.Errorf("dict: %v invalid key", k)
					}
					//val[k] = r.eval(x[1], v)
					val[k] = x[1]
				default:
					return nil, errors.Errorf("dict: %v invalid key", k)
				}
			} else {
				return nil, errors.Errorf("dict: %v invalid number of arg", x)
			}
		default:
			return nil, errors.Errorf("dict: %v invalid type of arg", x)
		}
	}
	return val, nil
}

func (r *Rule) keys_map(n []interface{}, v map[string]interface{}) (interface{}, error) {
	keys := []interface{}{}
	if len(n) != 1 {
		return nil, errors.New("keys: invalid number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("keys: %v", e)
	}
	switch x := x.(type) {
	case map[string]interface{}:
		for k, _ := range x {
			keys = append(keys, k)
		}
		return keys, nil
	}
	return nil, errors.Errorf("keys: %v is not a map", x)
}

func (r *Rule) has_key_map(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("has_key: wrong number of args")
	}
	k, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("has_key: invalid key: %v", e)
	}
	switch k := k.(type) {
	case string:
		keys, e := r.keys_map(n[1:], v)
		if e != nil {
			return nil, errors.Errorf("has_key: invalide data: %v", e)
		}
		if keys != nil {
			for _, k2 := range keys.([]interface{}) {
				if k2 == k {
					return true, nil
				}
			}
			return false, nil
		}
	}
	return nil, errors.Errorf("has_key: %v invalid key", k)

}

func (r *Rule) get(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("get: invalid number of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("get: %v", e)
	}
	switch x := x.(type) {
	case string:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("get: %v", e)
		}
		if y, ok := y.(map[string]interface{}); ok {
			return r.eval(y[x], v)
		} else {
			return nil, errors.Errorf("get: %v is not a map", y)
		}
	case int64:
		y, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("get: %v", e)
		}
		if y, ok := y.([]interface{}); ok {
			if x >= 0 && x < int64(len(y)) {
				return r.eval(y[x], v)
			} else {
				return nil, errors.Errorf("get: %v invalid index", x)
			}
		} else {
			return nil, errors.Errorf("get: %v is not an array", y)
		}
	case float64:
		if x == float64(int64(x)) {
			y, e := r.eval(n[1], v)
			if e != nil {
				return nil, errors.Errorf("get: %v", e)
			}
			if y, ok := y.([]interface{}); ok {
				if int64(x) >= 0 && int64(x) < int64(len(y)) {
					return r.eval(y[int64(x)], v)
				} else {
					return nil, errors.Errorf("get: %v invalide index", x)
				}
			} else {
				return nil, errors.Errorf("get: %v is not an array", y)
			}
		} else {
			return nil, errors.Errorf("get: %v invalid index", x)
		}
	}

	return nil, errors.Errorf("get: %v invalid arg", x)
}

func (r *Rule) array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	return r.eval(n, v)
}

func (r *Rule) proc(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return nil, errors.New("proc: no args")
	}
	var y interface{} = nil
	var e error
	for _, x := range n {
		y, e = r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("proc: %v", e)
		}
	}
	return y, nil
}

func (r *Rule) while_proc(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("while: invalid number of args")
	}

	var y interface{} = nil
	for true {
		b, e := r.boolean(n[0], v)
		if e != nil {
			return nil, errors.Errorf("while: %v", e)
		}
		if !b.(bool) {
			break
		}

		x, e := r.eval(n[1], v)
		if e != nil {
			return nil, errors.Errorf("while: %v", e)
		}

		y = x
	}

	return y, nil
}

func (r *Rule) len_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return 0, errors.New("len: no args")
	}
	if len(n) > 1 {
		return float64(int64(len(n))), nil
	}

	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("len: %v", e)
	}
	switch x := x.(type) {
	case []interface{}:
		return float64(int64(len(x))), nil
	case map[string]interface{}:
		return float64(int64(len(x))), nil

	default:
		return float64(1), nil
		/*
			y, e := r.eval(x, v)
			if (e != nil) {
				return nil, errors.Errorf("len: %v", e)
			}
			switch y := y.(type) {
			case []interface{}:
				return float64(int64(len(y))), nil
			case map[string]interface{}:
				return float64(int64(len(y))), nil
			}
		*/
	}
	return nil, errors.New("len: arg is not a map or array")
}

func (r *Rule) in_array(x interface{}, n []interface{}, v map[string]interface{}) bool {
	if len(n) == 0 {
		return false
	}
	switch x := x.(type) {
	case string:
		y, _ := r.eval(n[0], v)
		switch y := y.(type) {
		case string:
			if x == y {
				return true
			}
		}

	case int64:
		y, _ := r.eval(n[0], v)
		switch y := y.(type) {
		case int64:
			if x == y {
				return true
			}
		case float64:
			if float64(x) == y {
				return true
			}
		}

	case float64:
		y, _ := r.eval(n[0], v)
		switch y := y.(type) {
		case int64:
			if x == float64(y) {
				return true
			}
		case float64:
			if x == y {
				return true
			}
		}

	case bool:
		y, _ := r.eval(n[0], v)
		switch y := y.(type) {
		case bool:
			if x == y {
				return true
			}
		}
	}
	return r.in_array(x, n[1:], v)
}

func (r *Rule) in(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("in: invalid numbrer of args")
	}
	x, e := r.eval(n[0], v)
	if e != nil {
		return nil, errors.Errorf("in: %v", e)
	}
	switch x := x.(type) {
	case string:
		switch y := get64(n[1]).(type) {
		case string:
			return r.contains([]interface{}{y, x}, v)
		case []interface{}:
			return r.in_array(x, y, v), nil
		default:
			y, e := r.eval(y, v)
			if e != nil {
				return nil, errors.Errorf("in: %v", e)
			}
			switch y := y.(type) {
			case string:
				return r.contains([]interface{}{x, y}, v)
			case []interface{}:
				return r.in_array(x, y, v), nil
			}
			return nil, errors.Errorf("in: %v is not string or array type", y)
		}
	default:
		switch y := get64(n[1]).(type) {
		case []interface{}:
			return r.in_array(x, y, v), nil
		default:
			y, e := r.eval(y, v)
			if e != nil {
				return nil, errors.Errorf("in: %v", e)
			}
			switch y := y.(type) {
			case []interface{}:
				return r.in_array(x, y, v), nil
			}
			return nil, errors.Errorf("in: %v is not an array", y)
		}
	}
	return nil, errors.Errorf("in: %v invalid type", x)
}

func (r *Rule) map_array_eval(d []interface{}, f interface{}, val map[string]interface{}) (interface{}, error) {
	var tempv map[string]interface{}
	tempv = make(map[string]interface{})
	for k, v := range val {
		tempv[k] = v
	}
	result := []interface{}{}
	for _, v := range d {
		x, e := r.eval(v, val)
		if e != nil {
			return result, errors.Errorf("map: %v", e)
		}
		tempv["$current"] = x
		if x, ok := x.(map[string]interface{}); ok {
			for k1, v1 := range x {
				tempv[k1] = v1
			}
		}
		y, e := r.eval(f, tempv)
		if e != nil {
			return result, errors.Errorf("map: %v", e)
		}
		result = append(result, y)
	}
	return result, nil
}

func (r *Rule) map_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("map: invalid number of args")
	}

	switch x := get64(n[0]).(type) {
	case []interface{}:
		return r.map_array_eval(x, n[1], v)
	default:
		y, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("map: %v", e)
		}
		switch y := y.(type) {
		case []interface{}:
			return r.map_array_eval(y, n[1], v)
		}
	}
	return nil, errors.New("map: arg is not an array type")
}

func (r *Rule) all_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("all: invalid number of args")
	}

	var z interface{}
	var ze error
	num := 0

	switch x := get64(n[0]).(type) {
	case []interface{}:
		num = len(x)
		z, ze = r.filter_array_eval(x, n[1], v)
	default:
		y, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("all: %v", e)
		}
		switch y := y.(type) {
		case []interface{}:
			num = len(y)
			z, ze = r.filter_array_eval(y, n[1], v)
		default:
			return nil, errors.New("all: first arg is not an array type")
		}
	}

	if ze != nil {
		return nil, errors.Errorf("all: %v", ze)
	}

	if len(z.([]interface{})) == num {
		return true, nil
	} else {
		return false, nil
	}
}

func (r *Rule) some_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("some: invalid number of args")
	}

	var z interface{}
	var ze error

	switch x := get64(n[0]).(type) {
	case []interface{}:
		z, ze = r.filter_array_eval(x, n[1], v)
	default:
		y, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("some: %v", e)
		}
		switch y := y.(type) {
		case []interface{}:
			z, ze = r.filter_array_eval(y, n[1], v)
		default:
			return nil, errors.New("some: first arg is not an array type")
		}
	}

	if ze != nil {
		return nil, errors.Errorf("some: %v", ze)
	}

	if len(z.([]interface{})) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (r *Rule) none_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("none: invalid number of args")
	}

	var z interface{}
	var ze error

	switch x := get64(n[0]).(type) {
	case []interface{}:
		z, ze = r.filter_array_eval(x, n[1], v)
	default:
		y, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("none: %v", e)
		}
		switch y := y.(type) {
		case []interface{}:
			z, ze = r.filter_array_eval(y, n[1], v)
		default:
			return nil, errors.New("none: first arg is not an array type")
		}
	}

	if ze != nil {
		return nil, errors.Errorf("none: %v", ze)
	}

	if z == nil || len(z.([]interface{})) == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (r *Rule) filter_array_eval(d []interface{}, f interface{}, val map[string]interface{}) (interface{}, error) {
	var tempv map[string]interface{}
	tempv = make(map[string]interface{})
	for k, v := range val {
		tempv[k] = v
	}
	result := []interface{}{}
	for _, v := range d {
		currv, e := r.eval(v, val)
		if e != nil {
			return result, errors.Errorf("filter: %v", e)
		}
		tempv["$current"] = currv
		if currv, ok := currv.(map[string]interface{}); ok {
			for k1, v1 := range currv {
				tempv[k1] = v1
			}
		}
		b, e := r.boolean(f, tempv)
		if e != nil {
			return result, errors.Errorf("filter: %v", e)
		}
		if b, ok := b.(bool); ok {
			if b {
				result = append(result, currv)
			}
		} else {
			return nil, errors.Errorf("filter; %v is not bool type", b)
		}
	}
	return result, nil
}

func (r *Rule) filter_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 2 {
		return nil, errors.New("filter: invalid number of args")
	}

	switch x := get64(n[0]).(type) {
	case []interface{}:
		return r.filter_array_eval(x, n[1], v)
	default:
		y, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("filter: %v", e)
		}
		switch y := y.(type) {
		case []interface{}:
			return r.filter_array_eval(y, n[1], v)
		}
	}
	return nil, errors.New("filter: first arg is not an array type")
}

func (r *Rule) reduce_array_eval(d []interface{}, f interface{}, initv interface{}, val map[string]interface{}) (interface{}, error) {
	var tempv map[string]interface{}
	tempv = make(map[string]interface{})
	for k, v := range val {
		tempv[k] = v
	}
	x, e := r.eval(initv, val)
	if e != nil {
		return nil, errors.Errorf("reduce: %v", e)
	}

	tempv["$accumulator"] = x
	for _, v := range d {
		y, e := r.eval(v, val)
		if e != nil {
			return nil, errors.Errorf("reduce: %v", e)
		}
		tempv["$current"] = y
		if y, ok := y.(map[string]interface{}); ok {
			for k1, v1 := range y {
				tempv[k1] = v1
			}
		}
		z, e := r.eval(f, tempv)
		if e != nil {
			return nil, errors.Errorf("reduce: %v", e)
		}
		tempv["$accumulator"] = z
	}
	return tempv["$accumulator"], nil
}

func (r *Rule) reduce_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 3 {
		return nil, errors.New("reduce: wroing number of args")
	}

	switch x := get64(n[0]).(type) {
	case []interface{}:
		return r.reduce_array_eval(x, n[1], n[2], v)
	default:
		y, e := r.eval(x, v)
		if e != nil {
			return nil, errors.Errorf("reduce: %v", e)
		}
		switch y := y.(type) {
		case []interface{}:
			return r.reduce_array_eval(y, n[1], n[2], v)
		}
	}
	return nil, errors.New("reduce: first arg is not an array type")
}

func (r *Rule) merge_array(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) == 0 {
		return n, nil
	}

	m := []interface{}{}
	for _, x := range n {
		y, e := r.eval(x, v)
		if e != nil {
			return m, errors.Errorf("merge: %v", e)
		}
		if y == nil {
			continue
		}
		switch y := y.(type) {
		case []interface{}:
			if len(y) > 0 {
				m = append(m, y...)
			}
		default:
			m = append(m, y)
		}
	}
	return m, nil
}

func (r *Rule) log(n []interface{}, v map[string]interface{}) (interface{}, error) {
	if len(n) != 1 {
		return n, nil
	}

	x, e := r.eval(n[0], v)
	if e != nil {
		logger.Infof("log: %v", e)
	} else {
		logger.Infof("%v", x)
	}
	return x, e

}

// CombineRules accepts a list of rules and combines them into a single rule with the given operator.
func CombineRules(operator string, ruleComponents []map[string]interface{}) (*Rule, error) {
	var rulePointer *Rule = nil
	if utils.IsStringEmpty(operator) {
		errMsg := "Rule operator must be specified"
		logger.Error(errMsg)
		return rulePointer, errors.New(errMsg)
	}
	if len(ruleComponents) > 0 {
		predicate := R(operator)
		for _, ruleComponent := range ruleComponents {
			predicate[operator] = append(predicate[operator].([]interface{}), ruleComponent)
		}
		rule := NewRule(predicate)
		rulePointer = &rule
	}
	return rulePointer, nil
}
