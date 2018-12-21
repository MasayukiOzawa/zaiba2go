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

func getFileStats(sql *string) {
	type fileStruct struct {
		ServerName        string `db:"server_name" type:"tag"`
		SQLInstanceName   string `db:"sql_instance_name" type:"tag"`
		DatabaseName      string `db:"database_name" type:"tag"`
		FileDatabaseName  string `db:"file_database_name" type:"tag"`
		FileID            string `db:"file_id" type:"tag"`
		NumofReads        int64  `db:"num_of_reads" type:"field"`
		NumofBytesRead    int64  `db:"num_of_bytes_read" type:"field"`
		IoStallReadMs     int64  `db:"io_stall_read_ms" type:"field"`
		NumOfWrites       int64  `db:"num_of_writes" type:"field"`
		NumOfBytesWritten int64  `db:"num_of_bytes_written" type:"field"`
		IoStallWriteMs    int64  `db:"io_stall_write_ms" type:"field"`
		SizeOnDiskBytes   int64  `db:"size_on_disk_bytes" type:"field"`
	}
	fields := new(fileStruct)

	// クエリ実行
	rows, err := db.Queryx(*sql)
	if err != nil {
		fmt.Println(fmt.Sprintf("[%v] : [getFileStats] SQL Execution Error : %s\n", time.Now().Format(timeFormat), err.Error()))
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
			// fmt.Printf("%s : %s : %s \n", fi.Name, fi.Tag.Get("type"), rv.Field(i).Interface())
		}
		buf = append(buf, fmt.Sprintf("%s,%s,%s %s %d\n",
			"filestats",
			applicationIntent,
			string(tagValue),
			string(fieldValue),
			time.Now().UnixNano(),
		)...)
		// fmt.Println(string(tagValue))
		// fmt.Println(string(fieldValue))
		// fmt.Print(string(buf))
	}

	req, err := http.NewRequest("POST", influxdbURI, bytes.NewBuffer(buf))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("[%v] : [getFileStats] Response Status [%s]\n", time.Now().Format(timeFormat), resp.Status)
	}
	wg.Done()
}
