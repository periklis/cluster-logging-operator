package loki

const (
	lokiYaml = `
auth_enabled: false
ingester:
  chunk_idle_period: 3m
  chunk_block_size: 262144
  chunk_retain_period: 1m
  max_transfer_retries: 0
  lifecycler:
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
limits_config:
  enforce_metric_name: false
  ingestion_rate_mb: 32
  ingestion_burst_size_mb: 32

  reject_old_samples: false
  reject_old_samples_max_age: 336h
schema_config:
  configs:
  - from: 2018-04-15
    store: boltdb
    object_store: filesystem
    schema: v9
    index:
      prefix: index_
      period: 336h
server:
  http_listen_port: 3100
  grpc_server_max_recv_msg_size: 10485760
  grpc_server_max_send_msg_size: 10485760
storage_config:
  boltdb:
    directory: /data/loki/index
  filesystem:
    directory: /data/loki/chunks
chunk_store_config:
  max_look_back_period: 0
table_manager:
  retention_deletes_enabled: false
  retention_period: 0
    `

	lokiUtil = `
#!/bin/sh

HOSTNAME=$1
INDEX_NAME=$2

curl -G -s "http://${HOSTNAME}/loki/api/v1/query" --data-urlencode "query={index_name=\"${INDEX_NAME}\"}" --data-urlencode 'limit=5'
    `

    etcdPreStopCmd = `
EPS=""
for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
    EPS="${EPS}${EPS:+,}http://${SET_NAME}-${i}.${SET_NAME}:2379"
done

HOSTNAME=$(hostname)
AUTH_OPTIONS=""

member_hash() {
    etcdctl $AUTH_OPTIONS member list | grep http://${HOSTNAME}.${SET_NAME}:2380 | cut -d':' -f1 | cut -d'[' -f1
}

SET_ID=${HOSTNAME##*[^0-9]}

if [ "${SET_ID}" -ge ${INITIAL_CLUSTER_SIZE} ]; then
    echo "Removing ${HOSTNAME} from etcd cluster"
    ETCDCTL_ENDPOINT=${EPS} etcdctl $AUTH_OPTIONS member remove $(member_hash)
    if [ $? -eq 0 ]; then
        # Remove everything otherwise the cluster will no longer scale-up
        rm -rf /var/run/etcd/*
    fi
fi
`

    etcdCmd = `
HOSTNAME=$(hostname)
AUTH_OPTIONS=""
# store member id into PVC for later member replacement
collect_member() {
    while ! etcdctl $AUTH_OPTIONS member list &>/dev/null; do sleep 1; done
    etcdctl $AUTH_OPTIONS member list | grep http://${HOSTNAME}.${SET_NAME}:2380 | cut -d':' -f1 | cut -d'[' -f1 > /var/run/etcd/member_id
    exit 0
}

eps() {
    EPS=""
    for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
        EPS="${EPS}${EPS:+,}http://${SET_NAME}-${i}.${SET_NAME}:2379"
    done
    echo ${EPS}
}

member_hash() {
    etcdctl $AUTH_OPTIONS member list | grep http://${HOSTNAME}.${SET_NAME}:2380 | cut -d':' -f1 | cut -d'[' -f1
}

# we should wait for other pods to be up before trying to join
# otherwise we got "no such host" errors when trying to resolve other members
#for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
#    while true; do
#        echo "Waiting for ${SET_NAME}-${i}.${SET_NAME} to come up"
#        ping -W 1 -c 1 ${SET_NAME}-${i}.${SET_NAME} > /dev/null && break
#        sleep 1s
#    done
#done

# re-joining after failure?
if [ -e /var/run/etcd/default.etcd ]; then
    echo "Re-joining etcd member"
    member_id=$(cat /var/run/etcd/member_id)

    # re-join member
    ETCDCTL_ENDPOINT=$(eps) etcdctl $AUTH_OPTIONS member update ${member_id} http://${HOSTNAME}.${SET_NAME}:2380 | true
    exec etcd --name ${HOSTNAME} \
        --listen-peer-urls http://0.0.0.0:2380 \
        --listen-client-urls http://0.0.0.0:2379\
        --advertise-client-urls http://${HOSTNAME}.${SET_NAME}:2379 \
        --data-dir /var/run/etcd/default.etcd

fi

# etcd-SET_ID
SET_ID=${HOSTNAME##*[^0-9]}

# adding a new member to existing cluster (assuming all initial pods are available)
if [ "${SET_ID}" -ge ${INITIAL_CLUSTER_SIZE} ]; then
    export ETCDCTL_ENDPOINT=$(eps)

    # member already added?
    MEMBER_HASH=$(member_hash)
    if [ -n "${MEMBER_HASH}" ]; then
        # the member hash exists but for some reason etcd failed
        # as the datadir has not be created, we can remove the member
        # and retrieve new hash
        etcdctl $AUTH_OPTIONS member remove ${MEMBER_HASH}
    fi

    echo "Adding new member"
    etcdctl $AUTH_OPTIONS member add ${HOSTNAME} http://${HOSTNAME}.${SET_NAME}:2380 | grep "^ETCD_" > /var/run/etcd/new_member_envs

    if [ $? -ne 0 ]; then
        echo "Exiting"
        rm -f /var/run/etcd/new_member_envs
        exit 1
    fi

    cat /var/run/etcd/new_member_envs
    source /var/run/etcd/new_member_envs

    collect_member &

    exec etcd --name ${HOSTNAME} \
        --listen-peer-urls http://0.0.0.0:2380 \
        --listen-client-urls http://0.0.0.0:2379 \
        --advertise-client-urls http://${HOSTNAME}.${SET_NAME}:2379 \
        --data-dir /var/run/etcd/default.etcd \
        --initial-advertise-peer-urls http://${HOSTNAME}.${SET_NAME}:2380 \
        --initial-cluster ${ETCD_INITIAL_CLUSTER} \
        --initial-cluster-state ${ETCD_INITIAL_CLUSTER_STATE}

fi

PEERS=""
for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
    PEERS="${PEERS}${PEERS:+,}${SET_NAME}-${i}=http://${SET_NAME}-${i}.${SET_NAME}:2380"
done

collect_member &

# join member
exec etcd --name ${HOSTNAME} \
    --initial-advertise-peer-urls http://${HOSTNAME}.${SET_NAME}:2380 \
    --listen-peer-urls http://0.0.0.0:2380 \
    --listen-client-urls http://0.0.0.0:2379 \
    --advertise-client-urls http://${HOSTNAME}.${SET_NAME}:2379 \
    --initial-cluster-token etcd-cluster-1 \
    --initial-cluster ${PEERS} \
    --initial-cluster-state new \
    --data-dir /var/run/etcd/default.etcd
`

    lokiClusterConfig = `
auth_enabled: false
chunk_store_config:
  max_look_back_period: 0s
distributor:
  ring:
    kvstore:
      store: etcd
      etcd:
        endpoints:
        - loki-kvstore.openshift-logging.svc.cluster.local:2380
frontend:
  compress_responses: true
  max_outstanding_per_tenant: 200
frontend_worker:
  address: query-frontend.openshift-logging.svc.cluster.local:9095
  grpc_client_config:
    max_send_msg_size: 1.048576e+08
  parallelism: 8
ingester:
  chunk_block_size: 262144
  chunk_idle_period: 15m
  lifecycler:
    claim_on_rollout: true
    heartbeat_period: 5s
    interface_names:
      - eth0
    join_after: 30s
    num_tokens: 512
    ring:
      heartbeat_timeout: 1m
      kvstore:
        store: etcd
        etcd:
          endpoints:
          - loki-kvstore.openshift-logging.svc.cluster.local:2380
      replication_factor: 3
  max_transfer_retries: 60
ingester_client:
  grpc_client_config:
    max_recv_msg_size: 6.7108864e+07
  remote_timeout: 1s
limits_config:
  enforce_metric_name: false
  ingestion_burst_size_mb: 32
  ingestion_rate_mb: 32
  ingestion_rate_strategy: global
  max_global_streams_per_user: 10000
  max_query_length: 12000h
  max_query_parallelism: 32
  max_streams_per_user: 0
  reject_old_samples: true
  reject_old_samples_max_age: 168h
query_range:
  align_queries_with_step: true
  cache_results: true
  max_retries: 5
  split_queries_by_interval: 30m
schema_config:
  configs:
    - from: "2018-04-15"
      index:
        period: 168h
        prefix: loki_index_
      store: boltdb-shipper
      object_store: s3
      schema: v11
server:
  graceful_shutdown_timeout: 5s
  grpc_server_max_concurrent_streams: 1000
  grpc_server_max_recv_msg_size: 1.048576e+08
  grpc_server_max_send_msg_size: 1.048576e+08
  http_listen_port: 80
  http_server_idle_timeout: 120s
  http_server_write_timeout: 1m
storage_config:
  aws:
    s3: url-to-s3-here
  boltdb_shipper_config:
    active_index_directory: /data/loki/index
    shared_store_type: s3
    cache_location: /data/loki/index_cache
    resync_interval: 5s
table_manager:
  chunk_tables_provisioning:
    inactive_read_throughput: 0
    inactive_write_throughput: 0
    provisioned_read_throughput: 0
    provisioned_write_throughput: 0
  index_tables_provisioning:
    inactive_read_throughput: 0
    inactive_write_throughput: 0
    provisioned_read_throughput: 0
    provisioned_write_throughput: 0
  retention_deletes_enabled: false
  retention_period: 0s
`

    lokiClusterOverrides = `
overrides: {}
`
)
