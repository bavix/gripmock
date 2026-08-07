#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

COUNT="${BENCH_COUNT:-500000}"
STARTUP_RUNS="${BENCH_STARTUP_RUNS:-1000}"
CONCURRENCY_LEVELS="${BENCH_CONCURRENCY_LEVELS:-1,10,20,50}"
DURATION="${BENCH_DURATION:-30s}"
SWEEP_DURATION="${BENCH_SWEEP_DURATION:-10s}"
BAVIX_IMAGE="${BAVIX_IMAGE:-bavix/gripmock:3.18.3}"
TKPD_IMAGE="${TKPD_IMAGE:-tkpd/gripmock:v1.14}"
RESULTS="${BENCH_RESULTS_DIR:-results}"
NATIVE_BIN="${BENCH_NATIVE_BIN:-/tmp/gripmock-native}"
NATIVE_GRPC_PORT=45770
NATIVE_HTTP_PORT=45771
NATIVE_GATEWAY_PORT=45769

CPUS=2
MEMORY=4g

SERVICE=helloworld.Greeter

STUBS_FILE=stubs.json
CSV_FILE=names.csv
STARTUP_STUBS_FILE=stubs-startup.json

SCENARIOS="${BENCH_SCENARIOS:-hit miss}"

for bin in docker jq curl lsof grpctestify go; do
    command -v "$bin" >/dev/null || { echo "missing required tool: $bin" >&2; exit 1; }
done

mkdir -p "$RESULTS"

CURRENT_CID=""
NATIVE_PID=""
cleanup() {
    [[ -n "${NATIVE_PID:-}" ]] && kill "$NATIVE_PID" 2>/dev/null
    [[ -n "$CURRENT_CID" ]] && docker rm -f "$CURRENT_CID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "building native binary -> $NATIVE_BIN"
(cd .. && go build -o "$NATIVE_BIN" .)

echo "generating $COUNT stubs"
go run . gen "$COUNT" "$STUBS_FILE" "$CSV_FILE"
go run . gen 1 "$STARTUP_STUBS_FILE" /dev/null

now_epoch() { perl -MTime::HiRes=time -e 'printf "%.3f", time'; }

image_size_mb() {
    local image="$1" raw platforms arch digest bytes json="{}"

    raw="$(docker buildx imagetools inspect --raw "$image" 2>/dev/null)" || { echo '{}'; return; }
    platforms="$(jq -r '.manifests[]? | select(.platform.os=="linux")
        | select(.platform.architecture=="amd64" or .platform.architecture=="arm64")
        | "\(.platform.architecture) \(.digest)"' <<<"$raw")"
    [[ -z "$platforms" ]] && { echo '{}'; return; }

    while read -r arch digest; do
        bytes="$(docker buildx imagetools inspect --raw "$image@$digest" | jq '[.layers[].size] | add')"
        json="$(jq -c --arg a "$arch" --argjson mb "$(awk -v b="$bytes" 'BEGIN{printf "%.2f", b/1048576}')" \
            '. + {($a): $mb}' <<<"$json")"
    done <<<"$platforms"

    echo "$json"
}

wait_ready() {
    local port="$1" timeout="$2" waited=0
    until grpctestify reflect --address "127.0.0.1:$port" --plaintext 2>/dev/null \
        | grep -q "$SERVICE"; do
        sleep 0.05
        waited=$(awk -v w="$waited" 'BEGIN{printf "%.2f", w+0.05}')
        if awk -v w="$waited" -v t="$timeout" 'BEGIN{exit !(w>t)}'; then
            echo "timed out waiting for $SERVICE on gRPC port $port" >&2
            return 1
        fi
    done
}

start_native() {
    local stubs="${1:-$STUBS_FILE}" dir
    dir="$(mktemp -d)"
    cp "$stubs" "$dir/"

    lsof -ti "tcp:$NATIVE_GRPC_PORT" >/dev/null 2>&1 && { echo "port $NATIVE_GRPC_PORT busy" >&2; return 1; }

    GRPC_PORT="$NATIVE_GRPC_PORT" HTTP_PORT="$NATIVE_HTTP_PORT" \
    GATEWAY_PORT="$NATIVE_GATEWAY_PORT" GOMAXPROCS="$CPUS" \
        "$NATIVE_BIN" --stub="$dir" proto/service.proto >/dev/null 2>&1 &

    local pid=$!
    wait_ready "$NATIVE_GRPC_PORT" 300

    echo "$pid $NATIVE_GRPC_PORT $NATIVE_HTTP_PORT"
}

stop_native() {
    [[ -n "${NATIVE_PID:-}" ]] || return 0
    kill "$NATIVE_PID" 2>/dev/null || true
    wait "$NATIVE_PID" 2>/dev/null || true
    NATIVE_PID=""

    local waited=0
    while lsof -ti "tcp:$NATIVE_GRPC_PORT" >/dev/null 2>&1; do
        sleep 0.05
        waited=$(awk -v w="$waited" 'BEGIN{printf "%.2f", w+0.05}')
        if awk -v w="$waited" 'BEGIN{exit !(w>30)}'; then
            echo "native port $NATIVE_GRPC_PORT still held after stop" >&2
            return 1
        fi
    done
}

start_engine() {
    local image="$1" stubs="${2:-$STUBS_FILE}" cid grpc_port http_port
    cid="$(docker run -d --cpus="$CPUS" --memory="$MEMORY" \
        -p 127.0.0.1::4770 -p 127.0.0.1::4771 \
        -v "$(pwd)/proto:/proto:ro" \
        -v "$(pwd)/$stubs:/stub/$stubs:ro" \
        "$image" --stub=/stub /proto/service.proto)"

    grpc_port="$(docker port "$cid" 4770/tcp | head -1 | cut -d: -f2)"
    http_port="$(docker port "$cid" 4771/tcp | head -1 | cut -d: -f2)"

    wait_ready "$grpc_port" 300

    echo "$cid $grpc_port $http_port"
}

measure_startup() {
    local name="$1" image="$2" times cid grpc http start end i

    times="$(mktemp)"

    for ((i = 1; i <= STARTUP_RUNS; i++)); do
        start="$(now_epoch)"
        if [[ "$image" == native ]]; then
            read -r cid grpc http < <(start_native "$STARTUP_STUBS_FILE")
            NATIVE_PID="$cid"
            end="$(now_epoch)"
        else
            read -r cid grpc http < <(start_engine "$image" "$STARTUP_STUBS_FILE")
            CURRENT_CID="$cid"
            end="$(now_epoch)"
        fi

        awk -v s="$start" -v e="$end" 'BEGIN{printf "%.3f\n", e-s}' >>"$times"

        if [[ "$image" == native ]]; then
            stop_native
        else
            docker rm -f "$cid" >/dev/null
            CURRENT_CID=""
        fi

        if ((STARTUP_RUNS >= 10 && i % (STARTUP_RUNS / 10) == 0)); then
            echo "  [$name] startup $i/$STARTUP_RUNS" >&2
        fi
    done

    awk '{ if(NR==1||$1<min)min=$1; if(NR==1||$1>max)max=$1; sum+=$1; n++ }
         END{ printf "{\"min\":%.3f,\"avg\":%.3f,\"max\":%.3f,\"runs\":%d}", min, sum/n, max, n }' "$times"
    rm -f "$times"
}

verify_stubs() {
    local name="$1" port="$2" loaded
    if [[ "$name" == "tkpd" ]]; then
        loaded="$(curl -sf "http://127.0.0.1:$port/" | jq '[.[][]|length]|add')"
    else
        loaded="$(curl -sf "http://127.0.0.1:$port/api/stubs" | jq 'length')"
    fi

    if [[ "$loaded" != "$COUNT" ]]; then
        echo "[$name] expected $COUNT stubs loaded, admin API reports $loaded" >&2
        return 1
    fi
}

check_report() {
    local name="$1" scenario="$2" level="$3" report="$4" dist ok total

    if [[ ! -s "$report" ]]; then
        echo "[$name/$scenario] concurrency=$level: no benchmark report written" >&2
        return 1
    fi

    dist="$(jq -c '.error_distribution' "$report")"
    if [[ "$dist" != "{}" ]]; then
        echo "[$name/$scenario] concurrency=$level: transport failures: $dist" >&2
        return 1
    fi

    ok="$(jq '.summary.ok' "$report")"
    total="$(jq '.summary.count' "$report")"

    if [[ "$scenario" == "miss" ]]; then
        if [[ "$ok" != "0" ]]; then
            echo "[$name/$scenario] concurrency=$level: expected every request to miss, $ok/$total matched" >&2
            return 1
        fi
    elif [[ "$ok" != "$total" ]]; then
        echo "[$name/$scenario] concurrency=$level: expected every request to match, $ok/$total did" >&2
        return 1
    fi
}

run_engine() {
    local name="$1" image="$2" cid grpc http size_mb startup memory_mb
    local levels top level duration rps throughput scenario gctf

    echo "[$name] $image"
    if [[ "$image" == native ]]; then
        size_mb='{}'
    else
        size_mb="$(image_size_mb "$image")"
    fi

    startup="$(measure_startup "$name" "$image")"

    if [[ "$image" == native ]]; then
        read -r cid grpc http < <(start_native)
        NATIVE_PID="$cid"
        memory_mb="$(ps -o rss= -p "$cid" | awk '{printf "%.1f", $1/1024}')"
    else
        read -r cid grpc http < <(start_engine "$image")
        CURRENT_CID="$cid"
        memory_mb="$(docker stats --no-stream --format '{{.MemUsage}}' "$cid" \
            | sed 's| /.*||' \
            | awk '/GiB/{printf "%.1f", $0*1024} /MiB/{printf "%.1f", $0+0} /KiB/{printf "%.3f", $0/1024}')"
    fi
    echo "[$name] resident memory: ${memory_mb} MB"

    verify_stubs "$name" "$http"

    IFS=',' read -ra levels <<<"$CONCURRENCY_LEVELS"
    top="${levels[-1]}"

    for scenario in $SCENARIOS; do
        gctf="tests/$scenario.gctf"
        [[ -f "$gctf" ]] || { echo "no .gctf for scenario $scenario" >&2; return 1; }

        throughput="[]"

        for level in "${levels[@]}"; do
            duration="$SWEEP_DURATION"
            [[ "$level" == "$top" ]] && duration="$DURATION"

            echo "[$name/$scenario] grpctestify bench: concurrency=$level duration=$duration"
            GRPCTESTIFY_ADDRESS="127.0.0.1:$grpc" grpctestify bench "$gctf" \
                --duration "$duration" \
                --concurrency "$level" \
                --log-format json \
                --log-output "$RESULTS/$name-$scenario-c$level.json" >/dev/null || true

            if [[ "$image" == native ]]; then
                kill -0 "$cid" 2>/dev/null || { echo "[$name] process died mid-run" >&2; return 1; }
            else
                [[ "$(docker inspect -f '{{.State.Running}}' "$cid")" == "true" ]] \
                    || { echo "[$name] container stopped mid-run" >&2; return 1; }
            fi

            check_report "$name" "$scenario" "$level" "$RESULTS/$name-$scenario-c$level.json"

            rps="$(jq '.summary.rps_observed' "$RESULTS/$name-$scenario-c$level.json")"
            throughput="$(jq -c --argjson l "$level" --argjson r "$rps" \
                '. + [{concurrency: $l, rps: $r}]' <<<"$throughput")"
        done

        cp "$RESULTS/$name-$scenario-c$top.json" "$RESULTS/$name-$scenario.json"
        echo "$throughput" >"$RESULTS/$name-$scenario-throughput.json"
    done

    jq -n --arg image "$image" --argjson size_mb "$size_mb" \
        --argjson startup "$startup" --argjson memory_mb "$memory_mb" \
        '{image: $image, size_mb: $size_mb, startup: $startup, memory_mb: $memory_mb}' \
        >"$RESULTS/$name-meta.json"

    if [[ "$image" == native ]]; then
        stop_native
    else
        docker rm -f "$cid" >/dev/null
        CURRENT_CID=""
    fi
}

run_engine bavix "$BAVIX_IMAGE"
run_engine native native
run_engine tkpd "$TKPD_IMAGE"

echo "done -> $RESULTS/"
