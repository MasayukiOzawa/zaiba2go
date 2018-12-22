package main

func queryList() []string {
	return []string{queryPerfInfo, queryFileStats, queryCPU, queryMemory}
}

// パフォーマンスモニター
type perfStruct struct {
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

// ファイル I/O
type fileStruct struct {
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

// CPU
type cpuStruct struct {
	Measurement     string  `db:"measurement" type:"measurement"`
	ServerName      string  `db:"server_name" type:"tag"`
	SQLInstanceName string  `db:"sql_instance_name" type:"tag"`
	DatabaseName    string  `db:"database_name" type:"tag"`
	InstanceName    string  `db:"instance_name" type:"tag"`
	CPUUsage        float64 `db:"CPU_Usage" type:"field"`
}

const queryCPU = `
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

// メモリ
type memoryStruct struct {
	Measurement     string  `db:"measurement" type:"measurement"`
	ServerName      string  `db:"server_name" type:"tag"`
	SQLInstanceName string  `db:"sql_instance_name" type:"tag"`
	DatabaseName    string  `db:"database_name" type:"tag"`
	Type            string  `db:"type" type:"tag"`
	Name            string  `db:"name" type:"tag"`
	SizeKb          float64 `db:"size_kb" type:"field"`
}

const queryMemory = `
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
