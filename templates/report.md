# Cluster Utilization Report

```text
Organization              : {{ORGANIZATION}}
Dashboard                 : {{DASHBOARD}}
Date From                 : {{DATE_FROM}}
Date To                   : {{DATE_TO}}
Timezone                  : {{TIMEZONE}}
Generated At              : {{GENERATED_AT}}
```

---

## Cluster Information

```text
Type                      : {{CLUSTER_TYPE}}
Name                      : {{CLUSTER_NAME}}
Node Count                : {{NODE_COUNT}}
Data Centers              : {{DATA_CENTERS}}
Apache Cassandra Version  : {{CASSANDRA_VERSION}}
OS Version                : {{OS_VERSION}}
Java Version              : {{JAVA_VERSION}}
```

---

## Backups

{{BACKUPS_SECTION}}

---

## Security
{{SECURITY_SECTION}}

## Utilization

##### Max Disk Read Per Second
{{CHART_DISK_READ}}

##### Used Disk Space Per Node
{{CHART_DISK_USAGE}}

##### Average CPU Usage per DC
{{CHART_CPU}}

##### Max Disk Write Per Second
{{CHART_DISK_WRITE}}

##### Average Disk % Usage All
{{CHART_DISK_ALL_USAGE}}

## Coordinator

##### Coordinator Reads distribution
{{CHART_COORDINATOR_READS}}

##### Coordinator Writes distribution
{{CHART_COORDINATOR_WRITES}}

##### Coordinator Read Throughput Per $groupBy ($consistency) - Count Per Second
{{CHART_COORDINATOR_READ_THROUGHPUT}}

##### Total Coordinator Write Throughput Per $groupBy ($consistency) - Count Per Second
{{CHART_COORDINATOR_WRITE_THROUGHPUT}}

##### Max Coordinator Read $consistency Latency - $percentile
{{CHART_COORDINATOR_READ_LATENCY}}

##### Max Coordinator Write $consistency Latency - $percentile
{{CHART_COORDINATOR_WRITE_LATENCY}}