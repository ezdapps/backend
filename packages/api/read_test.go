// MIT License
//
// Copyright (c) 2016 GenesisCommunity
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package api

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	var (
		retCont contentResult
	)

	assert.NoError(t, keyLogin(1))

	name := randName(`tbl`)
	form := url.Values{"Name": {name}, "ApplicationId": {`1`},
		"Columns": {`[{"name":"my","type":"varchar", "index": "1", 
	  "conditions":"true"},
	{"name":"amount", "type":"number","index": "0", "conditions":"{\"update\":\"true\", \"read\":\"true\"}"},
	{"name":"active", "type":"character","index": "0", "conditions":"{\"update\":\"true\", \"read\":\"false\"}"}]`},
		"Permissions": {`{"insert": "true", "update" : "true", "read": "true", "new_column": "true"}`}}
	assert.NoError(t, postTx(`NewTable`, &form))

	contList := []string{`contract %s {
		action {
			DBInsert("%[1]s", "my,amount", "Alex", 100 )
			DBInsert("%[1]s", "my,amount", "Alex 2", 13300 )
			DBInsert("%[1]s", "my,amount", "Mike", 0 )
			DBInsert("%[1]s", "my,amount", "Mike 2", 25500 )
			DBInsert("%[1]s", "my,amount", "John Mike", 0 )
			DBInsert("%[1]s", "my,amount", "Serena Martin", 777 )
		}
	}`,
		`contract Get%s {
		action {
			var row array
			row = DBFind("%[1]s").Where("id>= ? and id<= ?", 2, 5)
		}
	}`,
		`contract GetOK%s {
		action {
			var row array
			row = DBFind("%[1]s").Columns("my,amount").Where("id>= ? and id<= ?", 2, 5)
		}
	}`,
		`contract GetData%s {
		action {
			var row array
			row = DBFind("%[1]s").Columns("active").Where("id>= ? and id<= ?", 2, 5)
		}
	}`,
		`func ReadFilter%s bool {
				var i int
				var row map
				while i < Len($data) {
					row = $data[i]
					if i == 1 || i == 3 {
						row["my"] = "No name"
						$data[i] = row
					}
					i = i+ 1
				}
				return true
			}`,
	}
	for _, contract := range contList {
		form = url.Values{"Value": {fmt.Sprintf(contract, name)}, "ApplicationId": {`1`},
			"Conditions": {`true`}}
		assert.NoError(t, postTx(`NewContract`, &form))
	}
	assert.NoError(t, postTx(name, &url.Values{}))

	assert.EqualError(t, postTx(`GetData`+name, &url.Values{}), `{"type":"panic","error":"Access denied"}`)
	assert.NoError(t, sendPost(`content`, &url.Values{`template`: {
		`DBFind(` + name + `, src).Limit(2)`}}, &retCont))

	if strings.Contains(RawToString(retCont.Tree), `active`) {
		t.Errorf(`wrong tree %s`, RawToString(retCont.Tree))
		return
	}

	assert.NoError(t, postTx(`GetOK`+name, &url.Values{}))

	assert.NoError(t, postTx(`EditColumn`, &url.Values{`TableName`: {name}, `Name`: {`active`},
		`Permissions`: {`{"update":"true", "read":"ContractConditions(\"MainCondition\")"}`}}))

	assert.NoError(t, postTx(`Get`+name, &url.Values{}))

	form = url.Values{"Name": {name}, "InsertPerm": {`ContractConditions("MainCondition")`},
		"UpdatePerm": {"true"}, "ReadPerm": {`false`}, "NewColumnPerm": {`true`}}
	assert.NoError(t, postTx(`EditTable`, &form))
	assert.EqualError(t, postTx(`GetOK`+name, &url.Values{}), `{"type":"panic","error":"Access denied"}`)

	form = url.Values{"Name": {name}, "InsertPerm": {`ContractConditions("MainCondition")`},
		"UpdatePerm": {"true"}, "FilterPerm": {`ReadFilter` + name + `()`},
		"NewColumnPerm": {`ContractConditions("MainCondition")`}}
	assert.NoError(t, postTx(`EditTable`, &form))

	var tableInfo tableResult
	assert.NoError(t, sendGet(`table/`+name, nil, &tableInfo))
	assert.Equal(t, `ReadFilter`+name+`()`, tableInfo.Filter)

	assert.NoError(t, sendPost(`content`, &url.Values{`template`: {
		`DBFind(` + name + `, src).Limit(2)`}}, &retCont))
	if !strings.Contains(RawToString(retCont.Tree), `No name`) {
		t.Errorf(`wrong tree %s`, RawToString(retCont.Tree))
		return
	}
}
