package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func execPerfInfo(sql *string) {
	type perfTags struct {
		servername      string
		sqlinstancename string
		databasename    string
		objectname      string
		countername     string
		instancename    string
		cntrtype        int
	}
	type perfKeys struct {
		cntrvalue int64
	}

	tags := new(perfTags)
	keys := new(perfKeys)

	// クエリ実行
	rows, err := db.Query(*sql)
	if err != nil {
		fmt.Println(fmt.Sprintf("[%v] : [execPerfInfo] SQL Execution Error : %s\n", time.Now().Format(timeFormat), err.Error()))
		wg.Done()
		return
	}

	buf := make([]byte, 0)
	for rows.Next() {
		// rows.Scan(&perf.servername, &perf.sqlinstancename, &perf.databasename, &perf.objectname, &perf.countername, &perf.instancename, &perf.cntrvalue, &perf.cntrtype)
		rows.Scan(
			&tags.servername,
			&tags.sqlinstancename,
			&tags.databasename,
			&tags.objectname,
			&tags.countername,
			&tags.instancename,
			&keys.cntrvalue,
			&tags.cntrtype,
		)

		buf = append(buf, fmt.Sprintf("%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%d,%s=%s %s=%d %d\n",
			strings.Replace(tags.objectname, " ", "\\ ", -1),
			"server_name", strings.Replace(tags.servername, " ", "\\ ", -1),
			"sql_instance_name", strings.Replace(tags.sqlinstancename, " ", "\\ ", -1),
			"database_name", strings.Replace(tags.databasename, " ", "\\ ", -1),
			"instance_name", strings.Replace(tags.instancename, " ", "\\ ", -1),
			"counter_name", strings.Replace(tags.countername, " ", "\\ ", -1),
			"cntr_type", tags.cntrtype,
			"application_intent", strings.Replace(*config.applicationintent, " ", "\\ ", -1),
			"cntr_value", keys.cntrvalue,
			time.Now().UnixNano(),
		)...)
	}
	req, err := http.NewRequest("POST", influxdbURI, bytes.NewBuffer(buf))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("[%v] : [execPerfInfo] Response Status [%s]\n", time.Now().Format(timeFormat), resp.Status)
	}
	wg.Done()
}
