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

	"github.com/BurntSushi/toml"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
)

const timeFormat = "2006-01-02 15:04:05.000"

var db *sqlx.DB
var wg sync.WaitGroup
var influxdbURI string
var config *Zaiba2Config
var sqlConfig tomlConfig

// Zaiba2Config : Config 用構造体 (NewZaiba2Config により生成する)
type Zaiba2Config struct {
	influxdbServer  *string
	influxdbPort    *int
	influxdbName    *string
	applicationname *string
	sleepinterval   *int
}

type structSQLConfig struct {
	ServerName        string
	UserID            string
	Password          string
	Database          string
	ApplicationIntent string
	AzureSQLDB        int
}

type tomlConfig struct {
	Server structSQLConfig
}

// Newzaiba2Config : 実行時引数を元に Config を作成
func Newzaiba2Config() *Zaiba2Config {
	// 実行時引数の取得
	config := new(Zaiba2Config)

	// 接続情報のアプリケーション名
	config.applicationname = flag.String("applicationname", "MSSQL Monitor Zaiba2", "Connected Application Name")

	// InfluxDB 接続情報
	config.influxdbServer = flag.String("influxdbServer", "localhost", "InfluxDb Server name")
	config.influxdbPort = flag.Int("influxdbPort", 8086, "InfluxdDb Port Number")
	config.influxdbName = flag.String("influxdbName", "zaiba2", "InfluxdDb DB Name")

	// 取得間隔
	config.sleepinterval = flag.Int("sleepinterval", 5, "Metrics Collect interval")

	flag.Parse()
	return config
}

func doMain() error {
	var err error
	config = Newzaiba2Config()

	// Ctrl + C で終了した場合の制御用チャネルの設定
	c := make(chan os.Signal)
	cStop := make(chan bool)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cStop <- true
	}()

	// InfluxDB の接続用 URI
	influxdbURI = fmt.Sprintf("http://%s:%d/write?db=%s",
		*config.influxdbServer,
		*config.influxdbPort,
		*config.influxdbName,
	)

	// TOML から SQL Server の接続を情報を読み込み
	_, err = toml.DecodeFile("zaiba2.config", &sqlConfig)

	if sqlConfig.Server.ApplicationIntent == "" {
		sqlConfig.Server.ApplicationIntent = "ReadWrite"
	}

	// 接続文字列の作成
	constring := fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;app name=%s;ApplicationIntent=%s",
		sqlConfig.Server.ServerName,
		sqlConfig.Server.UserID,
		sqlConfig.Server.Password,
		sqlConfig.Server.Database,
		*config.applicationname,
		sqlConfig.Server.ApplicationIntent,
	)

	// SQL Server 用のドライバーで初期化
	db, err = sqlx.Open("sqlserver", constring)
	if err != nil {
		return fmt.Errorf("SQL Open Error : %s", err.Error())
	}
	defer db.Close()

	// SQL Server への接続確認
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("SQL Ping Error : %s", err.Error())
	}

	// メトリクス取得前の初期設定
	query := queryList()
	runtime.GOMAXPROCS(runtime.NumCPU())

	// メトリクスの取得開始
	for {
		// fmt.Printf("[%v] : Metric Collect.\n", time.Now().Format(timeFormat))
		select {
		// Ctrl + C が押された場合の終了処理
		case <-cStop:
			fmt.Printf("[%v] : Received Stop Signal\n", time.Now().Format(timeFormat))
			wg.Wait()
			return nil
		default:

			wg.Add(len(query))

			go getMeasurement(&query[0], new(structPerfInfo))
			go getMeasurement(&query[1], new(structFileStats))
			go getMeasurement(&query[2], new(structCPUUsage))
			go getMeasurement(&query[3], new(structMemoryClerk))
			go getMeasurement(&query[4], new(structWorkerThread))
			go getMeasurement(&query[5], new(structWaitTask))
			go getMeasurement(&query[6], new(structWaitStats))
			go getMeasurement(&query[7], new(structTempdb))

			wg.Wait()

			time.Sleep(time.Second * time.Duration(*config.sleepinterval))
		}
	}
}

func main() {
	fmt.Printf("[%v] : Zaiba2 Start.\n", time.Now().Format(timeFormat))
	defer fmt.Printf("[%v] : Zaiba2 Stop.\n", time.Now().Format(timeFormat))

	err := doMain()

	if err != nil {
		log.Fatalf("[%v] : ERROR : %s\n", time.Now().Format(timeFormat), err.Error())
	}
}
