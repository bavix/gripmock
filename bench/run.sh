#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

COUNTS="${BENCH_COUNTS:-500 10000 500000}"
STARTUP_RUNS="${BENCH_STARTUP_RUNS:-50}"
CONCURRENCY_LEVELS="${BENCH_CONCURRENCY_LEVELS:-1,100}"
DURATION="${BENCH_DURATION:-30s}"
SWEEP_DURATION="${BENCH_SWEEP_DURATION:-10s}"  # also the cap for engines too slow to reach BENCH_REQUESTS
BAVIX_IMAGE="${BAVIX_IMAGE:-bavix/gripmock:3.18.3}"
TKPD_IMAGE="${TKPD_IMAGE:-tkpd/gripmock:v1.14}"
NATIVE_BIN="${BENCH_NATIVE_BIN:-/tmp/gripmock-native}"
NATIVE_GRPC_PORT=45770
NATIVE_HTTP_PORT=45771
NATIVE_GATEWAY_PORT=45769

CPUS="${BENCH_CPUS:-4}"
MEMORY="${BENCH_MEMORY:-8g}"
CONNECTIONS="${BENCH_CONNECTIONS:-8}"
REQUEST_TIMEOUT="${BENCH_REQUEST_TIMEOUT:-180s}"
# 300000 requests puts the fast engines in a ~12 s window, where repeated
# measurements of the same point stay within 2 %. At 50000 the window drops
# to 2 s and the spread grows to 10 %, which is noise, not signal.
REQUESTS="${BENCH_REQUESTS:-300000}"
WARMUP="${BENCH_WARMUP:-1s}"

SERVICE=helloworld.Greeter

STUBS_BASE=stubs
CSV_FILE=names.csv
STARTUP_BASE=stubs-startup
GEN_BIN=/tmp/gripmock-bench-gen

scenario_stubs() {
    case "$1" in
        contains) echo "$STUBS_BASE-contains.json" ;;
        matches)  echo "$STUBS_BASE-matches.json" ;;
        *)        echo "$STUBS_BASE-equals.json" ;;
    esac
}

scenario_gctf() {
    case "$1" in
        miss) echo "tests/miss.gctf" ;;
        *)    echo "tests/match.gctf" ;;
    esac
}

SCENARIOS="${BENCH_SCENARIOS:-equals contains matches miss}"

for bin in docker jq curl lsof grpctestify go perl; do
    command -v "$bin" >/dev/null || { echo "missing required tool: $bin" >&2; exit 1; }
done

CORES="$(getconf _NPROCESSORS_ONLN)"
if ((CORES < CPUS * 2)); then
    echo "need at least $((CPUS * 2)) cores to grant $CPUS to the engine and leave the rest to the load client; this host has $CORES" >&2
    echo "lower BENCH_CPUS to at most $((CORES / 2)) to proceed" >&2
    exit 1
fi

MAX_LOAD="${BENCH_MAX_LOAD:-1.5}"
host_load() {
    if [[ -r /proc/loadavg ]]; then
        cut -d' ' -f1 /proc/loadavg
    else
        uptime | sed 's/.*averages*: *//' | cut -d, -f1 | tr -d ' '
    fi
}

IDLE="$(host_load)"
if awk -v l="$IDLE" -v m="$MAX_LOAD" 'BEGIN{exit !(l > m)}'; then
    echo "host is busy: 1-minute load average is $IDLE, above BENCH_MAX_LOAD=$MAX_LOAD" >&2
    echo "another workload competing for CPU changes the numbers by tens of percent; wait for the host to settle or raise BENCH_MAX_LOAD to override" >&2
    exit 1
fi

CURRENT_CID=""
NATIVE_PID=""
cleanup() {
    [[ -n "${NATIVE_PID:-}" ]] && kill "$NATIVE_PID" 2>/dev/null
    [[ -n "$CURRENT_CID" ]] && docker rm -f "$CURRENT_CID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "building binaries"
(cd .. && go build -o "$NATIVE_BIN" .)
go build -o "$GEN_BIN" .

"$GEN_BIN" gen 1 "$STARTUP_BASE" /dev/null


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
    local stubs="$1" dir
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
    local image="$1" stubs="$2" cid grpc_port http_port
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
            read -r cid grpc http < <(start_native "$STARTUP_BASE-equals.json")
            NATIVE_PID="$cid"
            end="$(now_epoch)"
        else
            read -r cid grpc http < <(start_engine "$image" "$STARTUP_BASE-equals.json")
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

# sample_usage appends "cpu_percent rss_mb" once a second. Container CPU comes
# from docker stats, which is already an interval rate; the native process is
# read from /proc so the figure is the same kind of number -- `ps -o %cpu` on
# Linux reports an average over the process lifetime, which dilutes with idle
# time and is not comparable.
sample_usage() {
    local name="$1" target="$2" out="$3"
    : >"$out"

    if [[ "$name" == native ]]; then
        local ticks prev now delta rss
        ticks="$(getconf CLK_TCK)"
        prev="$(awk '{print $14 + $15}' "/proc/$target/stat" 2>/dev/null || echo 0)"

        while :; do
            sleep 1
            now="$(awk '{print $14 + $15}' "/proc/$target/stat" 2>/dev/null)" || break
            rss="$(awk '{printf "%.1f", $24 * 4096 / 1048576}' "/proc/$target/stat" 2>/dev/null)"
            delta="$(awk -v a="$prev" -v b="$now" -v t="$ticks" 'BEGIN{printf "%.1f", (b - a) * 100 / t}')"
            echo "$delta $rss" >>"$out"
            prev="$now"
        done

        return
    fi

    while :; do
        docker stats --no-stream --format '{{.CPUPerc}} {{.MemUsage}}' "$target" 2>/dev/null \
            | awk '{gsub(/%/,"",$1); mem=$2;
                    if (mem ~ /GiB/) {gsub(/GiB/,"",mem); mem*=1024}
                    else if (mem ~ /MiB/) {gsub(/MiB/,"",mem)}
                    else if (mem ~ /KiB/) {gsub(/KiB/,"",mem); mem/=1024}
                    printf "%.1f %.1f\n", $1, mem}' >>"$out"
        sleep 1
    done
}

# summarize_usage reports averages, not peaks. A peak read from `docker stats`
# is not a real quantity: it derives the percentage from its own sampling
# window, which does not line up with the 100 ms CFS quota periods, so short
# bursts read above the container's own CPU limit.
summarize_usage() {
    awk '{ csum += $1+0
           if ($2+0 > mem) mem = $2+0; msum += $2+0
           n++ }
         END { if (n == 0) { print "null"; exit }
               printf "{\"cpu_avg\":%.1f,\"mem_peak_mb\":%.1f,\"mem_avg_mb\":%.1f,\"samples\":%d}",
                      csum/n, mem, msum/n, n }' "$1"
}

verify_stubs() {
    local name="$1" port="$2" loaded
    if [[ "$name" == "tkpd" ]]; then
        loaded="$(curl -sf "http://127.0.0.1:$port/" | jq '[.[][]|length]|add')"
    else
        loaded="$(curl -sf -D - -o /dev/null "http://127.0.0.1:$port/api/stubs?limit=1" \
            | awk -F': *' 'tolower($1)=="x-total-count"{gsub(/\r/,"",$2); print $2}')"
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
    local name="$1" image="$2" cid grpc http size_mb startup memory_mb=null
    local levels top level duration rps throughput scenario gctf stubs connections
    local usage_raw sampler

    echo "[$name] $image"
    size_mb="${SIZE_CACHE[$name]}"

    if ((STARTUP_RUNS > 0)); then
        startup="$(measure_startup "$name" "$image")"
    else
        startup=null
    fi

    IFS=',' read -ra levels <<<"$CONCURRENCY_LEVELS"
    top="${levels[-1]}"

    for scenario in $SCENARIOS; do
        gctf="$(scenario_gctf "$scenario")"
        [[ -f "$gctf" ]] || { echo "missing $gctf for scenario $scenario" >&2; return 1; }

        stubs="$(scenario_stubs "$scenario")"

        if [[ "$image" == native ]]; then
            read -r cid grpc http < <(start_native "$stubs")
            NATIVE_PID="$cid"
        else
            read -r cid grpc http < <(start_engine "$image" "$stubs")
            CURRENT_CID="$cid"
        fi

        if [[ "$scenario" == equals ]]; then
            if [[ "$image" == native ]]; then
                memory_mb="$(ps -o rss= -p "$cid" | awk '{printf "%.1f", $1/1024}')"
            else
                memory_mb="$(docker stats --no-stream --format '{{.MemUsage}}' "$cid" \
                    | sed 's| /.*||' \
                    | awk '/GiB/{printf "%.1f", $0*1024} /MiB/{printf "%.1f", $0+0} /KiB/{printf "%.3f", $0/1024}')"
            fi
            echo "[$name] resident memory: ${memory_mb} MB"
        fi

        verify_stubs "$name" "$http"

        throughput="[]"

        usage_raw="$(mktemp)"

        for level in "${levels[@]}"; do
            duration="$SWEEP_DURATION"
            [[ "$level" == "$top" ]] && duration="$DURATION"

            connections=$((level < CONNECTIONS ? level : CONNECTIONS))

            echo "[$name/$scenario] grpctestify bench: concurrency=$level connections=$connections duration=$duration"

            sampler=""
            if [[ "$level" == "$top" ]]; then
                sample_usage "$image" "$cid" "$usage_raw" &
                sampler=$!
            fi
            GRPCTESTIFY_ADDRESS="127.0.0.1:$grpc" grpctestify bench "$gctf" \
                --requests "$REQUESTS" \
                --max-duration "$duration" \
                --warmup "$WARMUP" \
                --concurrency "$level" \
                --connections "$connections" \
                --request-timeout "$REQUEST_TIMEOUT" \
                --log-format json \
                --log-output "$RESULTS/$name-$scenario-c$level.json" >/dev/null

            if [[ "$image" == native ]]; then
                kill -0 "$cid" 2>/dev/null || { echo "[$name] process died mid-run" >&2; return 1; }
            else
                [[ "$(docker inspect -f '{{.State.Running}}' "$cid")" == "true" ]] \
                    || { echo "[$name] container stopped mid-run" >&2; return 1; }
            fi

            if [[ -n "$sampler" ]]; then
                kill "$sampler" 2>/dev/null || true
                wait "$sampler" 2>/dev/null || true
            fi

            check_report "$name" "$scenario" "$level" "$RESULTS/$name-$scenario-c$level.json"

            rps="$(jq '.summary.rps_observed' "$RESULTS/$name-$scenario-c$level.json")"
            throughput="$(jq -c --argjson l "$level" --argjson r "$rps" \
                '. + [{concurrency: $l, rps: $r}]' <<<"$throughput")"
        done

        summarize_usage "$usage_raw" >"$RESULTS/$name-$scenario-usage.json"
        rm -f "$usage_raw"

        cp "$RESULTS/$name-$scenario-c$top.json" "$RESULTS/$name-$scenario.json"
        echo "$throughput" >"$RESULTS/$name-$scenario-throughput.json"

        if [[ "$image" == native ]]; then
            stop_native
        else
            docker rm -f "$cid" >/dev/null
            CURRENT_CID=""
        fi
    done

    jq -n --arg image "$image" --argjson size_mb "$size_mb" \
        --argjson startup "$startup" --argjson memory_mb "$memory_mb" \
        --argjson cpus "$CPUS" --arg memory "$MEMORY" \
        '{image: $image, size_mb: $size_mb, startup: $startup, memory_mb: $memory_mb,
          limits: {cpus: $cpus, memory: $memory}}' \
        >"$RESULTS/$name-meta.json"
}

declare -A SIZE_CACHE=(
    [bavix]="$(image_size_mb "$BAVIX_IMAGE")"
    [native]='{}'
    [tkpd]="$(image_size_mb "$TKPD_IMAGE")"
)

STARTUP_RUNS_CONFIGURED="$STARTUP_RUNS"

for COUNT in $COUNTS; do
    RESULTS="results-$COUNT"
    mkdir -p "$RESULTS"

    echo "=== $COUNT stubs -> $RESULTS/"
    "$GEN_BIN" gen "$COUNT" "$STUBS_BASE" "$CSV_FILE"

    run_engine bavix "$BAVIX_IMAGE"
    run_engine native native
    run_engine tkpd "$TKPD_IMAGE"

    STARTUP_RUNS=0
done

STARTUP_RUNS="$STARTUP_RUNS_CONFIGURED"

echo "done -> results-*/"
