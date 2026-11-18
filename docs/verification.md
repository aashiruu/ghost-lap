# Verification Records and Execution Log

## Stage 0 Setup Verification
* Verified presence of exported JSON datasets in `data/RUS_lap.json` and `data/HAM_lap.json`.
* Sample counts verified: RUS = 688 samples, HAM = 689 samples.
* Average sampling rate verified: ~8.01 Hz.
## Stage 1 Producer Verification Log
* verified local Redpanda topic creation (`telemetry-raw`).
* Verified sample delivery order across drivers sorted by `rel_time_sec`.
* Verified speed scaling: 1380 total samples published to Redpanda in ~8.5s at 10x replay speed (`-speed=10.0`).
## Stage 2 Verification Log
* Executed `cmd/distance-check/main.go` on raw telemetry for RUS and HAM.
* Verified distance integration accumulation across all 690 telemetry samples per driver.
* Recorded total track distance and driver drift comparison in `docs/alignment-testing.md`.
## Stage 3 Verification Log
* Resampled telemetry streams for RUS and HAM onto a 1.0m uniform spatial grid.
* Verified physical boundary compliance across all grid points: Throttle $\in [0, 100]\%$, Brake $\in [0, 1.0]$.
## Stage 4: Delta & Ground-Truth Validation
* Command: `go run cmd/delta-check/main.go`
* Output: Computed finish-line delta `+0.1710 s` vs. official gap `+0.1710 s` (`0.0000 s` / `0.00 ms` error; status `VERIFIED`).
## Stage 5 Streaming Consumer Verification Log
* Ran incremental streaming simulation (`cmd/stream-check/main.go`).
* Verified equivalence between incremental streaming output and Stage 4 batch processing ($0.0000\text{ms}$ divergence).
## Stage 6 Verification Log
* Executed reproducible sample loss test harness (`cmd/degrade-check/main.go`).
* Measured finish line error and overall Mean Absolute Error (MAE) across 0%, 5%, 10%, and 25% drop rates.
## Stage 7 Segment Breakdown Verification Log
* Executed `cmd/segment-check/main.go` across 7 contiguous Silverstone segments.
* Verified segment telescoping sum equals `+0.1710 s` with `0.000000 ms` conservation error.
## Stage 8: Track Replay View
* Command: `go run cmd/replay-web/main.go -addr=:8080`
* Output: Exported `5,856` aligned replay frames to `web/replay_data.json` (`finish_delta: +0.1710 s`) and verified animated two-driver track rendering with live HUD delta readout in `web/index.html`.
## Stage 9: Observability (Prometheus & Grafana)
* Commands: `go build -o bin/observer cmd/observer/main.go` and `curl -s http://localhost:9090/metrics | grep '^ghostlap_'`
* Measured compiled pipeline pass: `processed 690 RUS + 690 HAM frames in 23.8878 ms (delta error: 0.000000 s)`
* Verified Prometheus metrics exposed at `GET /metrics`:
  * `ghostlap_frames_processed_total{driver="RUS"}` = `690`
  * `ghostlap_frames_processed_total{driver="HAM"}` = `690`
  * `ghostlap_consumer_lag_records{driver="RUS"}` = `0`
  * `ghostlap_consumer_lag_records{driver="HAM"}` = `0`
  * `ghostlap_delta_error_vs_official_seconds` = `0`
  * `ghostlap_resample_duration_seconds_sum` = `0.027466999` (`count = 1`, bucket `le="0.0512"` = `1`)
* Verified Grafana dashboard configuration (`deploy/grafana-dashboard.json`), Prometheus scrape target (`deploy/prometheus.yml`), and stack compose file (`deploy/docker-compose.yml`).
