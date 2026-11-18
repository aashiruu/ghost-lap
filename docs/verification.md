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
## Stage 5 Streaming Consumer Verification Log
* Ran incremental streaming simulation (`cmd/stream-check/main.go`).
* Verified equivalence between incremental streaming output and Stage 4 batch processing ($0.0000\text{ms}$ divergence).
## Stage 6 Verification Log
* Executed reproducible sample loss test harness (`cmd/degrade-check/main.go`).
* Measured finish line error and overall Mean Absolute Error (MAE) across 0%, 5%, 10%, and 25% drop rates.
