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

func getMeasurement(sql *string, fields interface{}) {
	// 構造体の名称 (処理対象) の取得
	typeNameSlice := strings.Split(reflect.TypeOf(fields).String(), ".")
	typeName := typeNameSlice[len(typeNameSlice)-1]

	// クエリ実行
	rows, err := db.Queryx(*sql)
	if err != nil {
		fmt.Println(fmt.Sprintf("[%v] : [%s] : SQL Execution Error : %s\n", time.Now().Format(timeFormat), typeName, err.Error()))
		wg.Done()
		return
	}
	buf := make([]byte, 0)

	for rows.Next() {
		var tagValue []byte
		var fieldValue []byte
		var measurement string

		rows.StructScan(fields)

		rt, rv := reflect.TypeOf(fields).Elem(), reflect.ValueOf(fields).Elem()

		for i := 0; i < rt.NumField(); i++ {
			fi := rt.Field(i)
			switch fi.Tag.Get("type") {
			case "measurement":
				measurement = strings.Replace(rv.Field(i).Interface().(string), " ", "\\ ", -1)
			case "tag":
				if tagValue == nil {
					tagValue = append(tagValue, (fi.Tag.Get("db") + "=" + strings.Replace(rv.Field(i).Interface().(string), " ", "\\ ", -1))...)
				} else {
					tagValue = append(tagValue, ("," + fi.Tag.Get("db") + "=" + strings.Replace(rv.Field(i).Interface().(string), " ", "\\ ", -1))...)
				}
			case "field":
				if fieldValue == nil {
					fieldValue = append(fieldValue, (strings.Replace(fi.Tag.Get("db"), " ", "\\ ", -1) + "=" + strconv.FormatFloat(rv.Field(i).Interface().(float64), 'f', 2, 64))...)
				} else {
					fieldValue = append(fieldValue, ("," + strings.Replace(fi.Tag.Get("db"), " ", "\\ ", -1) + "=" + strconv.FormatFloat(rv.Field(i).Interface().(float64), 'f', 2, 64))...)
				}
			}
		}
		buf = append(buf, fmt.Sprintf("%s,%s,%s %s %d\n",
			measurement,
			applicationIntent,
			string(tagValue),
			string(fieldValue),
			time.Now().UnixNano(),
		)...)
	}

	req, err := http.NewRequest("POST", influxdbURI, bytes.NewBuffer(buf))
	// fmt.Println(string(buf))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[%v] : [%s] : Post Failed [%s]\n", time.Now().Format(timeFormat), typeName, err.Error())
	} else if resp.StatusCode != 204 {
		fmt.Printf("[%v] : [%s] : HTTP Request Failed, Response Status [%s]\n", time.Now().Format(timeFormat), typeName, resp.Status)
	}
	wg.Done()
}
