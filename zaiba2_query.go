package main

func queryList() []string {

	var query []string
	// Azure SQL DB 向けのクエリセット
	if *config.azuresqldb == true {
		query = []string{
			queryPerfInfo,
			queryFileStats,
			queryCPUUsage,
			queryMemoryClerk,
			queryWorkerThreadAzure,
			queryWaitTask,
			queryWaitStats,
			queryTempdb,
		}
	} else {
		query = []string{
			queryPerfInfo,
			queryFileStats,
			queryCPUUsage,
			queryMemoryClerk,
			queryWorkerThread,
			queryWaitTask,
			queryWaitStats,
			queryTempdb,
		}
	}
	return query
}

/*************************************************************************/
// パフォーマンスモニター
type structPerfInfo struct {
	Measurement     string  `db:"measurement" type:"measurement"`
	ServerName      string  `db:"server_name" type:"tag"`
	SQLInstanceName string  `db:"sql_instance_name" type:"tag"`
	DatabaseName    string  `db:"database_name" type:"tag"`
	CounterName     string  `db:"counter_name" type:"tag"`
	InstanceName    string  `db:"instance_name" type:"tag"`
	CntrType        string  `db:"cntr_type" type:"tag"`
	CntrValue       float64 `db:"cntr_value" type:"field"`
}

const queryPerfInfo = `
SELECT 
	* 
FROM
(
	SELECT
		@@SERVERNAME AS server_name,
		COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
		DB_NAME() AS database_name,
		RTRIM(
			SUBSTRING(
			object_name, 
			PATINDEX('%:%', object_name) + 1,
			LEN(object_name) - PATINDEX('%:%', object_name) 
			)) AS measurement,
		RTRIM(counter_name) AS counter_name,
		CASE instance_name
			WHEN '' THEN ' '
			ELSE RTRIM(instance_name) 
		END AS instance_name,
		cntr_value,
		cntr_type
	FROM 
		sys.dm_os_performance_counters WITH(NOLOCK)
) AS T
WHERE 
	measurement IN('SQL Statistics', 'Buffer Manager', 'General Statistics', 'Locks', 'SQL Errors', 'Access Methods', 'Databases', 'HTTP Storage', 'Broker/DBM Transport', 'Database Replica', 'Availability Replica')
OPTION (RECOMPILE, MAXDOP 1);
`

/*************************************************************************/
// ファイル I/O
type structFileStats struct {
	Measurement       string  `db:"measurement" type:"measurement"`
	ServerName        string  `db:"server_name" type:"tag"`
	SQLInstanceName   string  `db:"sql_instance_name" type:"tag"`
	DatabaseName      string  `db:"database_name" type:"tag"`
	FileDatabaseName  string  `db:"file_database_name" type:"tag"`
	FileID            string  `db:"file_id" type:"tag"`
	NumofReads        float64 `db:"num_of_reads" type:"field"`
	NumofBytesRead    float64 `db:"num_of_bytes_read" type:"field"`
	IoStallReadMs     float64 `db:"io_stall_read_ms" type:"field"`
	NumOfWrites       float64 `db:"num_of_writes" type:"field"`
	NumOfBytesWritten float64 `db:"num_of_bytes_written" type:"field"`
	IoStallWriteMs    float64 `db:"io_stall_write_ms" type:"field"`
	SizeOnDiskBytes   float64 `db:"size_on_disk_bytes" type:"field"`
}

const queryFileStats = `
SELECT
	*
FROM
(SELECT
	'filestats' AS measurement,
    @@SERVERNAME AS server_name,
	COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
    DB_NAME() AS database_name,
	DB_NAME(database_id) AS file_database_name,
	file_id,
	num_of_reads,
	num_of_bytes_read,
	io_stall_read_ms,
	num_of_writes,
	num_of_bytes_written,
	io_stall_write_ms,
	size_on_disk_bytes
FROM 
	sys.dm_io_virtual_file_stats(NULL, NULL)) AS T1 
WHERE
	file_database_name IS NOT NULL
ORDER BY
	file_database_name ASC, file_id ASC
OPTION (RECOMPILE, MAXDOP 1);
`

/*************************************************************************/
// CPU 使用状況
type structCPUUsage struct {
	Measurement     string  `db:"measurement" type:"measurement"`
	ServerName      string  `db:"server_name" type:"tag"`
	SQLInstanceName string  `db:"sql_instance_name" type:"tag"`
	DatabaseName    string  `db:"database_name" type:"tag"`
	InstanceName    string  `db:"instance_name" type:"tag"`
	CPUUsage        float64 `db:"CPU_Usage" type:"field"`
}

const queryCPUUsage = `
SELECT
	'cpustats' AS measurement,
	server_name,
	sql_instance_name,
	database_name,
	instance_name,
	CAST([CPU usage %] AS float) / CAST([CPU usage % base] AS float) AS [CPU_Usage]
FROM
(
SELECT
    @@SERVERNAME AS server_name,
	COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
    DB_NAME() AS database_name,
	RTRIM(
		SUBSTRING(
		object_name, 
		PATINDEX('%:%', object_name) + 1,
		LEN(object_name) - PATINDEX('%:%', object_name) 
		)) AS object_name,
	RTRIM(counter_name) AS counter_name,
	CASE instance_name
		WHEN '' THEN ' '
		ELSE RTRIM(instance_name) 
	END AS instance_name,
	cntr_value
FROM 
	sys.dm_os_performance_counters WITH(NOLOCK)
WHERE 
	object_name LIKE '%Workload Group Stats%'
	AND
	counter_name IN('CPU usage %','CPU usage % base')
) AS T
PIVOT(
	SUM(cntr_value)
	FOR counter_name IN ([CPU usage %], [CPU usage % base])
) AS PV
OPTION(RECOMPILE, MAXDOP 1);
`

/*************************************************************************/
// メモリクラーク
type structMemoryClerk struct {
	Measurement     string  `db:"measurement" type:"measurement"`
	ServerName      string  `db:"server_name" type:"tag"`
	SQLInstanceName string  `db:"sql_instance_name" type:"tag"`
	DatabaseName    string  `db:"database_name" type:"tag"`
	Type            string  `db:"type" type:"tag"`
	Name            string  `db:"name" type:"tag"`
	SizeKb          float64 `db:"size_kb" type:"field"`
}

const queryMemoryClerk = `
DECLARE @ProductVersion nvarchar(128) = CAST(SERVERPROPERTY('ProductVersion') AS nvarchar(128))
DECLARE @MajorVersion int = (SELECT SUBSTRING(@ProductVersion, 1, CHARINDEX('.', @ProductVersion) - 1))
DECLARE @sql nvarchar(max) = '
SELECT 
	''memoryclerk'' AS measurement,
	@@SERVERNAME AS server_name,
	COALESCE(SERVERPROPERTY(''InstanceName''), ''MSSQLSERVER'') AS sql_instance_name,
	DB_NAME() AS database_name,
	type, 
	name,
	SUM(%%pages%% + awe_allocated_kb) AS size_kb
FROM 
	sys.dm_os_memory_clerks WITH(NOLOCK)
GROUP BY 
	type,
	name
OPTION (RECOMPILE, MAXDOP 1);
'

IF @MajorVersion <= 10
BEGIN
	SET @sql = REPLACE(@sql, '%%pages%%' , 'single_pages_kb + multi_pages_kb')
END
ELSE
BEGIN
	SET @sql = REPLACE(@sql, '%%pages%%' , 'pages_kb')
END

EXECUTE(@sql);
`

/*************************************************************************/
// ワーカースレッド
type structWorkerThread struct {
	Measurement         string  `db:"measurement" type:"measurement"`
	ServerName          string  `db:"server_name" type:"tag"`
	SQLInstanceName     string  `db:"sql_instance_name" type:"tag"`
	DatabaseName        string  `db:"database_name" type:"tag"`
	CurrentTasksCount   float64 `db:"current_tasks_count" type:"field"`
	RunnableTasksCount  float64 `db:"runnable_tasks_count" type:"field"`
	CurrentWorkersCount float64 `db:"current_workers_count" type:"field"`
	ActiveWorkersCount  float64 `db:"active_workers_count" type:"field"`
	WorkQueueCount      float64 `db:"work_queue_count" type:"field"`
	MaxWorkersCount     float64 `db:"max_workers_count" type:"field"`
}

const queryWorkerThread = `
SELECT
	'workerthread' AS measurement,
	@@SERVERNAME AS server_name,
	COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
	DB_NAME() AS database_name,
	SUM(current_tasks_count) AS current_tasks_count,
	SUM(runnable_tasks_count) AS runnable_tasks_count,
	SUM(current_workers_count) AS current_workers_count,
	SUM(active_workers_count) AS active_workers_count,
	SUM(work_queue_count) AS work_queue_count,
	(SELECT max_workers_count FROM sys.dm_os_sys_info) AS max_workers_count
FROM 
	sys.dm_os_schedulers WITH(NOLOCK)
WHERE 
	status = 'VISIBLE ONLINE'
OPTION (RECOMPILE, MAXDOP 1);
`
const queryWorkerThreadAzure = `
SELECT
	'workerthread' AS measurement,
	@@SERVERNAME AS server_name,
	COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
	DB_NAME() AS database_name,
	SUM(current_tasks_count) AS current_tasks_count,
	SUM(runnable_tasks_count) AS runnable_tasks_count,
	SUM(current_workers_count) AS current_workers_count,
	SUM(active_workers_count) AS active_workers_count,
	SUM(work_queue_count) AS work_queue_count,
	0 AS max_workers_count
FROM 
	sys.dm_os_schedulers WITH(NOLOCK)
WHERE 
	status = 'VISIBLE ONLINE'
OPTION (RECOMPILE, MAXDOP 1);
`

/*************************************************************************/
// Wait Task
type structWaitTask struct {
	Measurement               string  `db:"measurement" type:"measurement"`
	ServerName                string  `db:"server_name" type:"tag"`
	SQLInstanceName           string  `db:"sql_instance_name" type:"tag"`
	DatabaseName              string  `db:"database_name" type:"tag"`
	SessionID                 string  `db:"session_id" type:"tag"`
	Status                    string  `db:"status" type:"tag"`
	Command                   string  `db:"command" type:"tag"`
	WaitType                  string  `db:"wait_type" type:"tag"`
	HostName                  string  `db:"host_name" type:"tag"`
	ProgramName               string  `db:"program_name" type:"tag"`
	ElapsedTimeSec            float64 `db:"elapsed_time_sec" type:"field"`
	TransactionElapsedTimeSec float64 `db:"transaction_elapsed_time_sec" type:"field"`
}

const queryWaitTask = `
SELECT
	'waittask' AS measurement,
	@@SERVERNAME AS server_name,
	COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
	DB_NAME() AS database_name, 
	wt.session_id,
	COALESCE(er.status, ' ') AS status,
	COALESCE(er.command, ' ') AS command,
	COALESCE(er.wait_type, ' ') AS wait_type,
	COALESCE(es.host_name, ' ') AS host_name,
	COALESCE(es.program_name, ' ') AS program_name,
	COALESCE(datediff(SECOND,  er.start_time, GETDATE()), 0) AS elapsed_time_sec,
	CASE at.transaction_type
		WHEN 2 THEN 0
		ELSE COALESCE(datediff(SECOND, at.transaction_begin_time, GETDATE()), 0)
	END AS transaction_elapsed_time_sec
FROM
	sys.dm_os_waiting_tasks AS wt WITH(NOLOCK)
	LEFT JOIN sys.dm_exec_requests AS er WITH(NOLOCK) ON wt.session_id = er.session_id
	LEFT JOIN sys.dm_tran_active_transactions AS at WITH(NOLOCK) ON at.transaction_id = er.transaction_id
	LEFT JOIN sys.dm_exec_sessions AS es WITH(NOLOCK) ON es.session_id = er.session_id
WHERE
	wt.session_id > 0
ORDER BY
	wt.session_id
OPTION (RECOMPILE, MAXDOP 1)
`

/*************************************************************************/
// Wait Stats
type structWaitStats struct {
	Measurement       string  `db:"measurement" type:"measurement"`
	ServerName        string  `db:"server_name" type:"tag"`
	SQLInstanceName   string  `db:"sql_instance_name" type:"tag"`
	DatabaseName      string  `db:"database_name" type:"tag"`
	WaitCategory      string  `db:"wait_category" type:"tag"`
	WaitTimeMs        float64 `db:"wait_time_ms" type:"field"`
	WaitingTasksCount float64 `db:"waiting_tasks_count" type:"field"`
	MaxWaitTimeMs     float64 `db:"max_wait_time_ms" type:"field"`
}

const queryWaitStats = `
;WITH waitcategorystats ( 
	wait_category, wait_type, wait_time_ms, 
    waiting_tasks_count, 
    max_wait_time_ms) 
AS (
	SELECT 
		CASE 
			WHEN wait_type LIKE 'LCK%' THEN 'LOCKS' 
			WHEN wait_type LIKE 'PAGEIO%' THEN 'PAGE I/O LATCH' 
			WHEN wait_type LIKE 'PAGELATCH%' THEN 'PAGE LATCH (non-I/O)' 
			WHEN wait_type LIKE 'LATCH%' THEN 'LATCH (non-buffer)' 
			WHEN wait_type LIKE 'LATCH%' THEN 'LATCH (non-buffer)' 
			ELSE wait_type 
		END AS wait_category, 
		wait_type, 
		wait_time_ms, 
		waiting_tasks_count, 
		max_wait_time_ms 
	FROM   
		sys.dm_os_wait_stats WITH(NOLOCK)
    WHERE  
		wait_type NOT IN ( 
			'LAZYWRITER_SLEEP', 'CLR_AUTO_EVENT', 'CLR_MANUAL_EVENT' ,
			'REQUEST_FOR_DEADLOCK_SEARCH', 'BACKUPTHREAD', 'CHECKPOINT_QUEUE', 
			'EXECSYNC', 'FFT_RECOVERY', 
			'SNI_CRITICAL_SECTION', 'SOS_PHYS_PAGE_CACHE', 
			'CXROWSET_SYNC', 'DAC_INIT', 'DIRTY_PAGE_POLL', 
			'PWAIT_ALL_COMPONENTS_INITIALIZED', 'MSQL_XP', 'WAIT_FOR', 
			'DBMIRRORING_CMD', 'DBMIRROR_DBM_EVENT', 'DBMIRROR_EVENTS_QUEUE', 
			'DBMIRROR_WORKER_QUEUE', 'XE_TIMER_EVENT', 'XE_DISPATCHER_WAIT', 
			'WAITFOR_TASKSHUTDOWN', 'WAIT_FOR_RESULTS', 
			'SQLTRACE_INCREMENTAL_FLUSH_SLEEP', 'WAITFOR' ,'QDS_CLEANUP_STALE_QUERIES_TASK_MAIN_LOOP_SLEEP' ,
			'QDS_PERSIST_TASK_MAIN_LOOP_SLEEP', 'HADR_FILESTREAM_IOMGR_IOCOMPLETION', 
			'LOGMGR_QUEUE', 'FSAGENT' ) 
		AND wait_type NOT LIKE 'PREEMPTIVE%' 
		AND wait_type NOT LIKE 'SQLTRACE%' 
		AND wait_type NOT LIKE 'SLEEP%' 
		AND wait_type NOT LIKE 'FT_%' 
		AND wait_type NOT LIKE 'XE%' 
		AND wait_type NOT LIKE 'BROKER%' 
		AND wait_type NOT LIKE 'DISPATCHER%' 
		AND wait_type NOT LIKE 'PWAIT%' 
		AND wait_type NOT LIKE 'SP_SERVER%'
) 
SELECT 
		'waitstats' AS measurement,
        @@SERVERNAME AS server_name,
        COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
        DB_NAME() AS database_name, 
        wait_category, 
        Sum(wait_time_ms)        AS wait_time_ms, 
        Sum(waiting_tasks_count) AS waiting_tasks_count, 
        Max(max_wait_time_ms)    AS max_wait_time_ms 
FROM   waitcategorystats 
WHERE  wait_time_ms > 1000 
GROUP  BY wait_category  
OPTION (RECOMPILE, MAXDOP 1);
`

/*************************************************************************/
// tempdb
type structTempdb struct {
	Measurement                  string  `db:"measurement" type:"measurement"`
	ServerName                   string  `db:"server_name" type:"tag"`
	SQLInstanceName              string  `db:"sql_instance_name" type:"tag"`
	DatabaseName                 string  `db:"database_name" type:"tag"`
	FileName                     string  `db:"file_name" type:"tag"`
	UnallocatedExtentPageMB      float64 `db:"unallocated_extent_page_mb" type:"field"`
	VersionstoreReservedPageMB   float64 `db:"version_store_reserved_page_mb" type:"field"`
	UserobjectReservedPageMB     float64 `db:"user_object_reserved_page_mb" type:"field"`
	InternalObjectReservedPageMB float64 `db:"internal_object_reserved_page_mb" type:"field"`
	MixedExtentPageMB            float64 `db:"mixed_extent_page_mb" type:"field"`
}

const queryTempdb = `
SELECT
	'tempdb' AS measurement,
    @@SERVERNAME AS server_name,
    COALESCE(SERVERPROPERTY('InstanceName'), 'MSSQLSERVER') AS sql_instance_name,
    DB_NAME() AS database_name, 
	FILE_NAME([file_id]) AS [file_name],
	/*
	[total_page_count] * 8 / 1024 AS total_page_mb,
	[allocated_extent_page_count] * 8 / 1024 AS allocated_extent_page_mb,
	[modified_extent_page_count] * 8 / 1024 AS modified_extent_page_mb,
	*/
	[unallocated_extent_page_count] * 8 / 1024 AS unallocated_extent_page_mb,
	[version_store_reserved_page_count] * 8 / 1024 AS version_store_reserved_page_mb,
	[user_object_reserved_page_count] * 8 / 1024 AS [user_object_reserved_page_mb],
	[internal_object_reserved_page_count] * 8 / 1024 AS internal_object_reserved_page_mb,
	[mixed_extent_page_count] * 8 / 1024 AS mixed_extent_page_mb
FROM
	[tempdb].[sys].[dm_db_file_space_usage] WITH(NOLOCK)
OPTION (RECOMPILE, MAXDOP 1);
`
