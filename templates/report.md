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