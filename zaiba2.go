package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
)

const timeFormat = "2006-01-02 15:04:05.000"

var db *sqlx.DB
var wg sync.WaitGroup
var influxdbURI string
var config *Zaiba2Config
var applicationIntent string

type zaiba2Error struct {
	message string
}

func (e zaiba2Error) Error() string {
	return e.message
}

// Zaiba2Config : Config 用構造体 (NewZaiba2Config により生成する)
type Zaiba2Config struct {
	server            *string
	userID            *string
	password          *string
	database          *string
	influxdbServer    *string
	influxdbPort      *int
	influxdbName      *string
	applicationintent *string
}

// Newzaiba2Config : 実行時引数を元に Config を作成
func Newzaiba2Config() *Zaiba2Config {
	// 実行時引数の取得
	config := new(Zaiba2Config)
	config.server = flag.String("server", "localhost", "SQL Server Server Name")
	config.userID = flag.String("userid", "", "Login User Name")
	config.password = flag.String("password", "", "Login Password")
	config.database = flag.String("database", "master", "Connect Database")
	config.influxdbServer = flag.String("influxdbServer", "localhost", "InfluxDb Server name")
	config.influxdbPort = flag.Int("influxdbPort", 8086, "InfluxdDb Port Number")
	config.influxdbName = flag.String("influxdbName", "zaiba2", "InfluxdDb DB Name")
	config.applicationintent = flag.String("applicationintent", "ReadWrite", "ApplicationIntent")

	flag.Parse()
	return config
}

func doMain() error {
	var err error
	var zaiba2Err zaiba2Error
	config = Newzaiba2Config()

	// Ctrl + C で終了した場合の制御用チャネルの設定
	c := make(chan os.Signal)
	c1 := make(chan bool)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		c1 <- true
	}()

	// 接続文字列の作成
	constring := fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;",
		*config.server,
		*config.userID,
		*config.password,
		*config.database,
	)

	influxdbURI = fmt.Sprintf("http://%s:%d/write?db=%s",
		*config.influxdbServer,
		*config.influxdbPort,
		*config.influxdbName,
	)

	// SQL Server 用のドライバーで初期化
	db, err = sqlx.Open("sqlserver", constring)
	if err != nil {
		zaiba2Err.message = fmt.Sprintf("SQL Open Error : %s", err.Error())
		return zaiba2Err
	}
	defer db.Close()

	// SQL Server への接続確認
	err = db.Ping()
	if err != nil {
		zaiba2Err.message = fmt.Sprintf("SQL Ping Error : %s", err.Error())
		return zaiba2Err
	}

	// メトリクスの取得
	query := queryList()
	runtime.GOMAXPROCS(runtime.NumCPU())
	applicationIntent = "application_intent=" + (*config.applicationintent)

	for {
		select {
		// Ctrl + C が押された場合の終了処理
		case <-c1:
			fmt.Printf("[%v] : Received Stop Signal\n", time.Now().Format(timeFormat))
			wg.Wait()
			return nil
		default:
			wg.Add(len(query))
			go getMeasurement(&query[0], new(perfStruct))
			go getMeasurement(&query[1], new(fileStruct))
			go getMeasurement(&query[2], new(cpuStruct))
			go getMeasurement(&query[3], new(memoryStruct))

			wg.Wait()
			time.Sleep(time.Second * 5)
		}
	}
}

func main() {
	fmt.Printf("[%v] : Zaiba2 Start\n", time.Now().Format(timeFormat))
	defer fmt.Printf("[%v] : Zaiba2 Stop\n", time.Now().Format(timeFormat))

	err := doMain()

	if err != nil {
		log.Fatalf("[%v] : ERROR : %s\n", time.Now().Format(timeFormat), err.Error())
	}
}
