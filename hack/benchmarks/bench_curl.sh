#!/usr/bin/env bash
# Requires: bash, curl, awk, sort, GNU date
set -u

BASE_URL="${BASE_URL:-http://localhost:8080}"
BASE_URL="${BASE_URL%/}"
REQUESTS="${REQUESTS:-10000}"       # Total requests, split ~50/50 per endpoint
CONCURRENCY="${CONCURRENCY:-20}"
CONNECT_TIMEOUT="${CONNECT_TIMEOUT:-2}"
MAX_TIME="${MAX_TIME:-10}"

[[ "$REQUESTS" =~ ^[1-9][0-9]*$ && "$CONCURRENCY" =~ ^[1-9][0-9]*$ ]] || {
  echo "REQUESTS and CONCURRENCY must be positive integers" >&2
  exit 2
}

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

worker() {
  local worker="$1" i endpoint url query result
  : > "$workdir/$worker.tsv"

  for ((i=worker; i<REQUESTS; i+=CONCURRENCY)); do
    if (( i % 2 == 0 )); then
      endpoint="/collab/search"
      query="$(printf '%04x%04x%04x%04x' "$RANDOM" "$RANDOM" "$RANDOM" "$RANDOM")"
      url="$BASE_URL$endpoint?q=$query"
    else
      endpoint="/collab"
      url="$BASE_URL$endpoint"
    fi

    # Result fields: HTTP status, total request time in seconds
    if result="$(curl -sS \
      --connect-timeout "$CONNECT_TIMEOUT" \
      --max-time "$MAX_TIME" \
      -o /dev/null \
      -w '%{http_code}\t%{time_total}' \
      "$url" 2>/dev/null)"; then
      printf '%s\t%s\n' "$endpoint" "$result" >> "$workdir/$worker.tsv"
    else
      printf '%s\t000\t0\n' "$endpoint" >> "$workdir/$worker.tsv"
    fi
  done
}

start_ns="$(date +%s%N)"

pids=()
for ((w=0; w<CONCURRENCY; w++)); do
  worker "$w" &
  pids+=("$!")
done

for pid in "${pids[@]}"; do
  wait "$pid"
done

end_ns="$(date +%s%N)"

cat "$workdir"/*.tsv > "$workdir/results.tsv"
elapsed="$(awk -v a="$start_ns" -v b="$end_ns" 'BEGIN { printf "%.3f", (b-a)/1e9 }')"

percentile_ms() {
  awk -v p="$1" '
    { v[NR] = $1 }
    END {
      if (!NR) { print "n/a"; exit }
      i = int((p * NR + 99) / 100)
      if (i < 1) i = 1
      printf "%.2f", v[i] * 1000
    }
  ' "$2"
}

report() {
  local endpoint="$1" name="$2" stats
  local lat="$workdir/$name.lat"
  local total ok err avg p50 p95 p99

  awk -F '\t' -v ep="$endpoint" '$1 == ep && $2 != "000" { print $3 }' \
    "$workdir/results.tsv" | sort -n > "$lat"

  stats="$(awk -F '\t' -v ep="$endpoint" '
    $1 == ep {
      total++
      if ($2 ~ /^[23][0-9][0-9]$/) ok++; else err++
      if ($2 != "000") { sum += $3; measured++ }
    }
    END {
      printf "%d %d %d %.2f\n", total, ok, err,
        measured ? (sum / measured) * 1000 : 0
    }
  ' "$workdir/results.tsv")"

  read -r total ok err avg <<< "$stats"
  p50="$(percentile_ms 50 "$lat")"
  p95="$(percentile_ms 95 "$lat")"
  p99="$(percentile_ms 99 "$lat")"

  echo
  echo "=== $endpoint ==="
  printf 'Requests:               %s\n' "$total"
  printf 'Success (2xx/3xx):      %s\n' "$ok"
  printf 'Errors:                 %s\n' "$err"
  printf 'Latency avg/p50/p95/p99: %s / %s / %s / %s ms\n' \
    "$avg" "$p50" "$p95" "$p99"

  echo "Status codes:"
  awk -F '\t' -v ep="$endpoint" '
    $1 == ep { count[$2]++ }
    END { for (c in count) print c, count[c] }
  ' "$workdir/results.tsv" |
    sort -n |
    awk '{ printf "  %s: %s\n", $1, $2 }'
}

summary="$(awk -F '\t' '
  {
    total++
    if ($2 ~ /^[23][0-9][0-9]$/) ok++
    else err++
  }
  END { print total+0, ok+0, err+0 }
' "$workdir/results.tsv")"

read -r total ok err <<< "$summary"
rps="$(awk -v n="$total" -v s="$elapsed" 'BEGIN { printf "%.2f", n/s }')"

echo "=== Overall ==="
printf 'Requests / concurrency: %s / %s\n' "$REQUESTS" "$CONCURRENCY"
printf 'Wall time:              %s s\n' "$elapsed"
printf 'Throughput:             %s req/s\n' "$rps"
printf 'Success / errors:       %s / %s\n' "$ok" "$err"

report "/collab" "collab"
report "/collab/search" "search"