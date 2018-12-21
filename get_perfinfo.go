package main

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func getPerfinfo(sql *string) {
	type perfStruct struct {
		ServerName      string `db:"server_name" type:"tag"`
		SQLInstanceName string `db:"sql_instance_name" type:"tag"`
		DatabaseName    string `db:"database_name" type:"tag"`
		ObjectName      string `db:"object_name" type:"measurement"`
		CounterName     string `db:"counter_name" type:"tag"`
		InstanceName    string `db:"instance_name" type:"tag"`
		CntrType        string `db:"cntr_type" type:"tag"`
		CntrValue       int64  `db:"cntr_value" type:"field"`
	}
	fields := new(perfStruct)

	// クエリ実行
	rows, err := db.Queryx(*sql)
	if err != nil {
		fmt.Println(fmt.Sprintf("[%v] : [getPerfinfo] SQL Execution Error : %s\n", time.Now().Format(timeFormat), err.Error()))
		wg.Done()
		return
	}

	buf := make([]byte, 0)

	for rows.Next() {
		var tagValue []byte
		var fieldValue []byte

		rows.StructScan(fields)
		rt, rv := reflect.TypeOf(*fields), reflect.ValueOf(*fields)

		for i := 0; i < rt.NumField(); i++ {
			fi := rt.Field(i)
			switch fi.Tag.Get("type") {
			case "tag":
				if tagValue == nil {
					tagValue = append(tagValue, (fi.Tag.Get("db") + "=" + strings.Replace(rv.Field(i).Interface().(string), " ", "\\ ", -1))...)
				} else {
					tagValue = append(tagValue, ("," + fi.Tag.Get("db") + "=" + strings.Replace(rv.Field(i).Interface().(string), " ", "\\ ", -1))...)
				}
			case "field":
				if fieldValue == nil {
					fieldValue = append(fieldValue, (fi.Tag.Get("db") + "=" + strconv.FormatInt(rv.Field(i).Interface().(int64), 10))...)
				} else {
					fieldValue = append(fieldValue, ("," + fi.Tag.Get("db") + "=" + strconv.FormatInt(rv.Field(i).Interface().(int64), 10))...)
				}
			}
		}

		buf = append(buf, fmt.Sprintf("%s,%s,%s %s %d\n",
			strings.Replace(fields.ObjectName, " ", "\\ ", -1),
			tagValue,
			applicationIntent,
			fieldValue,
			time.Now().UnixNano(),
		)...)
	}

	req, err := http.NewRequest("POST", influxdbURI, bytes.NewBuffer(buf))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("[%v] : [getPerfinfo] Response Status [%s]\n", time.Now().Format(timeFormat), resp.Status)
	}
	wg.Done()
}
