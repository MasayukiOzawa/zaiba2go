package main

// 実行するクエリの定義
const queryPerfInfo = `
SELECT 
	* 
FROM
(
	SELECT
		@@SERVERNAME AS servername,
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
		cntr_value,
		cntr_type
	FROM 
		sys.dm_os_performance_counters WITH(NOLOCK)
) AS T
WHERE 
	object_name IN('SQL Statistics', 'Buffer Manager', 'General Statistics', 'Locks', 'SQL Errors', 'Access Methods', 'Databases', 'HTTP Storage', 'Broker/DBM Transport', 'Database Replica', 'Availability Replica')
OPTION (RECOMPILE, MAXDOP 1);
`

const queryFileStats = `
SELECT
	*
FROM
(SELECT
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

func queryList() []string {
	return []string{queryPerfInfo}
}
