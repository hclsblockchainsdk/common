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
	"fmt"
)

// Operators supported by simple rule:
//
//        (Math operations)
//        +
//        -
//        *
//        /
//        %
//        max
//        min
//        int
//        float
//        round  (round toward 0)
//        floor
//        ceil
//        sqrt
//
//        (String operations)
//        +  (same as cat)
//        cat
//        max
//        min
//        int
//        float
//        contains
//        in
//        substr
//
//        (Logical, Boolean and Comparison operations)
//        ==
//        ===
//        !=
//        !==
//        >
//        >=
//        <
//        <=
//        between (using < and <=)
//        !  (same as not)
//        not
//        !!  (same as bool)
//        bool
//        and
//        &&  (same as and)
//        or
//        ||  (same as or)
//        if
//
//        (Array and Dictionary operations)
//        array
//        list  (same as array)
//        in
//        for_each
//        map  (same as for_each)
//        filter
//        reduce
//        merge
//        dict
//        len
//        keys
//        has_key
//        get
//
//        (Accessing Data operations)
//        var
//        missing
//        missing_some
//        :=  (same as let)
//        let
//        proc
//        (Miscellaneous)
//        log
//
// Their behavior of these operators is standard json logic. A caller can combine rule expressions
// together by nesting them inside each other.  When the var operator is used, it is operating
// against the global_data.Asset structure which is like this:
//
// type Asset struct {
//     AssetId        string            `json:"asset_id"`
//     Datatypes      []string          `json:"datatypes"`
//     PublicData     []byte            `json:"public_data"`
//     PrivateData    []byte            `json:"private_data"`
//     OwnerIds       []string          `json:"owner_ids"`
//     Metadata       map[string]string `json:"metadata"`
//     AssetKeyId     string            `json:"asset_key_id"`
//     AssetKeyHash   []byte            `json:"asset_key_hash"`
//     IndexTableName string            `json:"index_table_name"`
// }
//
// In the example below, the rule is accessing the asset_id field and performing some evaluation on it.
// Caller can access any of these fields plus any nested fields. here, the vehicle object is stored entirely
// in private data.
//
// rule := simple_rule.NewRule(simple_rule.R("!=",
//     simple_rule.R("var", "private_data.vehicle_id"),
//     "vehicle1"),
// )

func ExampleNewRule() {
	rule1 := NewRule(`{"+": [1, 2.4, 6]}`)
	fmt.Println(rule1)

	e := make(map[string]interface{})
	e["+"] = []interface{}{1, 2.4, 6}
	rule2 := NewRule(e)
	fmt.Println(rule2)

	rule3 := NewRule(R("+", 1, 2.4, 6))
	fmt.Println(rule3)

	// Output: {map[+:[1 2.4 6]] map[]}
	// {map[+:[1 2.4 6]] map[]}
	// {map[+:[1 2.4 6]] map[]}
}

func ExampleR() {
	r := R("+", 1, 2.4, 6)
	fmt.Println(r)

	// Output: map[+:[1 2.4 6]]
}

func ExampleD() {
	normalizedDataSingle := D(1)
	fmt.Println(normalizedDataSingle)

	normalizedDataMulti := D(1, "a", "b")
	fmt.Println(normalizedDataMulti)

	// Output: 1
	// [1 a b]
}

func ExampleA() {
	normalizedDataSingle := A(1)
	fmt.Println(normalizedDataSingle)

	normalizedDataMulti := A(1, "a", "b")
	fmt.Println(normalizedDataMulti)

	// Output: [1]
	// [1 a b]
}

func ExampleM() {
	M(D("name", "Tom"), D("age", 32), D("sex", "male"))
	// returns map[name:Tom age:32 sex:male]
}

func ExampleRule_GetExpr() {
	rule := NewRule(R("+", 1, 2.4, 6))

	expr := rule.GetExpr()
	fmt.Println(expr)

	// Output: map[+:[1 2.4 6]]
}

func ExampleRule_GetExprJSON() {
	rule := NewRule(R("+", 1, 2.4, 6))

	exprJSON := rule.GetExprJSON()
	fmt.Println(exprJSON)

	// Output: {"+":[1,2.4,6]}
}

func ExampleRule_GetInit() {
	init_data := M(D("my list", D(1, 2, 3, 4.0, "my val", 1, "test")))
	rule := NewRule(R("in", "my val", R("var", "my list")), init_data)

	init := rule.GetInit()

	fmt.Println(init)
	// Output: map[my list:[1 2 3 4 my val 1 test]]
}

func ExampleRule_GetInitJSON() {
	init_data := M(D("my list", D(1, 2, 3, 4.0, "my val", 1, "test")))
	rule := NewRule(R("in", "my val", R("var", "my list")), init_data)

	initJSON := rule.GetInitJSON()

	fmt.Println(initJSON)
	// Output: {"my list":[1,2,3,4,"my val",1,"test"]}
}

func ExampleRule_Apply() {
	data := `{
		"name": {"last": "Smith", "first": "Jo"},
		"age": 45
	  }`
	rule := NewRule(R("var", "age"))
	resultMap, _ := rule.Apply(data)

	fmt.Println(resultMap["$result"])
	// Output: 45
}

func ExampleToJSON() {
	data := `{
		"name": {"last": "Smith", "first": "Jo"},
		"age": 45
	  }`
	rule := NewRule(R("var", "age"))
	resultMap, _ := rule.Apply(data)

	resultJSON := ToJSON(resultMap)

	fmt.Println(resultJSON)
	// Output: {"$result":45,"age":45,"name":{"first":"Jo","last":"Smith"}}
}
