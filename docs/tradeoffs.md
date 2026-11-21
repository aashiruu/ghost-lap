# Architectural Trade-Offs and Decisions

## Stage 0: Telemetry Ingestion and Serialization
* **Decision**: Offline JSON export via OpenF1 REST endpoints (`/v1/sessions`, `/v1/laps`, `/v1/car_data`, `/v1/location`) with a `0.5 s` boundary pad around each driver's fastest Q3 lap (`session_key=9554`), trimmed to `[0.0 s, official_lap_time]` via linear boundary interpolation (`TrimToOfficialLapWindow`).
* **Trade-Off**: Eliminates upstream rate-limiting and network jitter during pipeline testing while preserving exact lap start/finish boundary anchoring (`RUS`: `324` trimmed samples; `HAM`: `335` trimmed samples from `338` buffered samples).
* **Data Granularity**: OpenF1 `car_data` and `location` sample asynchronously at `~3.8 Hz` (`3.74 Hz` for `RUS`, `3.88 Hz` for `HAM`). Non-uniform sample intervals are preserved during ingestion and resampled onto a uniform spatial grid in Stage 3.

## Stage 1 & Stage 5: Redpanda Replay Producer & Consumer
* **Decision**: Single topic (`telemetry-raw`) keyed by driver code (`RUS`, `HAM`) with configurable time-delta replay scaling in `cmd/producer` and incremental stateful consumption (`DriverStreamState`) in `cmd/consumer`.
* **Trade-Off**: Emitting frame-by-frame over a local Redpanda broker adds ~1–2 ms of socket overhead compared to in-memory slices, but enforces partition ordering and verifies that incremental streaming produces identical `1.0 m` grid points (`0.0000 ms` divergence) to batch processing.

## Stage 2: Track Distance Alignment Trade-Offs
* **Speed Integration ($\int v \, dt$) vs. 2D Arc-Length Projection ($X, Y \to S$)**:
  * **Speed Integration**: Converts `km/h` to `m/s` (`/ 3.6`) and integrates over sample `dt`, yielding `5828.112 m` (`RUS`) and `5825.070 m` (`HAM`) with `3.042 m` inter-driver drift from trapezoidal approximation across `~260 ms` intervals.
  * **2D Arc-Length Projection**: Converts OpenF1 `location` coordinates from decimeters to meters (`math.Hypot(dx, dy) / 10.0`), yielding `5828.725 m` (`RUS`) and `5823.350 m` (`HAM`). The `5.376 m` difference reflects a combination of racing-line variation and `~3.8 Hz` position sampling across the 5.891 km Silverstone circuit.

## Stage 3: Distance Grid Resampling and Interpolation
* **Linear vs. Cubic Spline Interpolation**:
  * **Linear Interpolation**: Adopted for streaming resampling (`ResampleLinear`). Enforces strict monotonicity on `RelTimeSec`, prevents overshoot on step inputs (brake on/off, throttle snaps), and executes in $O(1)$ amortized time per `1.0 m` step.
  * **Cubic Spline**: Smooths continuous curves but introduces ringing/overshoot across step transitions unless heavily clamped.

## Stage 4: Uncalibrated Delta vs. Official Ground Truth
* **Decision**: Compute `DeltaSec = time_ham(S) - time_rus(S)` directly from each sample's interpolated timestamp and arc-length distance without finish-line tail calibration.
* **Trade-Off**: Avoids circular validation (`0.0000 s` forced error). Comparing both drivers on the overlapping `0–5,823 m` grid stops at the end of `HAM`'s shorter `5,823.35 m` trace (`85.984 s` vs. `85.990 s` official, `6 ms` short), while `RUS` still has `5.7 m` remaining on his `5,828.73 m` trace (`85.728 s` vs. `85.819 s` official, `91 ms` short)—producing a net `+0.2560 s` uncalibrated finish delta (`84.99 ms` off the `+0.1710 s` official gap).

## Stage 6: Sample-Loss Degradation Testing
* **Decision**: Seeded PRNG (`seed = 42`) drops intermediate telemetry samples at `5%`, `10%`, and `25%` while anchoring lap start and finish boundary samples.
* **Trade-Off**: Isolates interior interpolation degradation (`3.87 ms`, `13.29 ms`, and `52.41 ms` MAE vs. the `0%` baseline) using a single fixed seed for reproducibility.

## Stage 7: Distance-Band Segmentation
* **Decision**: Partition the overlapping `5,823 m` grid into 7 contiguous distance bands (`Band 1`–`Band 7`) and verify average speed bounds (`35–90 m/s`) on all bands longer than `600 m` without assigning unverified corner labels.
* **Trade-Off**: Avoids mislabeling corners without brake-zone alignment while preserving exact telescoping conservation (`0.000000 ms` difference between band sum and total uncalibrated delta).

## Stage 8 & Stage 9: Canvas Replay & Bounded-Cardinality Observability
* **Decision**: Serve static replay frames (`web/replay_data.json`) over Go `net/http` (`:8080`), expose strictly bounded enum-labeled Prometheus metrics (`driver` $\in \{\text{RUS}, \text{HAM}\}$, cardinality 2) on `GET /metrics`, and serve high-resolution `1.0 m` trace arrays on `GET /api/trace` (`:9090`).
* **Trade-Off**: Prevents Prometheus TSDB high-cardinality series explosion while keeping full `1.0 m` trace resolution available for Grafana panels.
