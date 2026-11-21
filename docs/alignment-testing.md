# Spatial Alignment and Validation Proofs

## 1. Distance Alignment Method Comparison (2024 Silverstone Q3, Session `9554`)
| Alignment Method | RUS Computed Distance | HAM Computed Distance | Difference |
| :--- | :--- | :--- | :--- |
| Speed Integration ($\int v \, dt$, `km/h` $\to$ `m/s`) | `5828.112 m` | `5825.070 m` | `3.042 m` |
| 2D Arc-Length Projection ($X, Y$ decimeters $\to$ `m`) | `5828.725 m` | `5823.350 m` | `5.376 m` |

* **Analysis**: Trimming both drivers strictly to their official Q3 lap windows (`RUS` Lap 25: `85.819 s`, `324` samples; `HAM` Lap 24: `85.990 s`, `335` trimmed samples) places both totals within ~1% of Silverstone's official `5,891 m` centerline. The `5.376 m` arc-length difference reflects racing-line variance and `~3.8 Hz` position sampling.

## 2. Uncalibrated Ground-Truth Lap-Time Delta Validation
* **Official Ground-Truth Gap**: `+0.1710 s` (`RUS 85.819 s` vs. `HAM 85.990 s`)
* **Computed Finish-Line Delta (Uncalibrated, at `5823 m`)**: `+0.2560 s`
* **Absolute Error vs. Official**: `0.0850 s` (`84.99 ms`)
* **Analysis**: Both drivers are compared up to `5,823 m`, the end of `HAM`'s shorter `5,823.35 m` position trace (`85.984 s` vs. `85.990 s` official, `6 ms` short). Because `RUS`'s trace is `5,828.73 m` long, comparing him at `5,823 m` stops `5.7 m` before his finish line (`85.728 s` vs. `85.819 s` official, `91 ms` short), yielding a net `85 ms` difference.

## 3. Sample-Loss Degradation Test (PRNG Seed = 42, Endpoints Anchored)
| Loss Rate | Finish Delta (s) | Error vs. Official (ms) | Max Curve Error vs. Base (ms) | MAE vs. Base (ms) |
| :--- | :--- | :--- | :--- | :--- |
| `0%` (Baseline) | `+0.2560 s` | `84.99 ms` | `0.00 ms` | `0.00 ms` |
| `5%` Loss | `+0.2594 s` | `88.37 ms` | `131.89 ms` | `3.87 ms` |
| `10%` Loss | `+0.2588 s` | `87.82 ms` | `139.23 ms` | `13.29 ms` |
| `25%` Loss | `+0.3066 s` | `135.63 ms` | `217.24 ms` | `52.41 ms` |

## 4. Distance-Band Breakdown (2024 Silverstone Q3)
| Distance Band | Length (m) | RUS (s) | HAM (s) | RUS Avg (m/s) | HAM Avg (m/s) | Delta (s) | Ahead |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `Band 1 (0-780 m)` | `780` | `9.518` | `9.581` | `81.95` | `81.41` | `+0.0625` | `RUS` |
| `Band 2 (780-1560 m)` | `780` | `15.474` | `15.482` | `50.41` | `50.38` | `+0.0072` | `RUS` |
| `Band 3 (1560-2520 m)` | `960` | `15.884` | `15.775` | `60.44` | `60.86` | `-0.1095` | `HAM` |
| `Band 4 (2520-3180 m)` | `660` | `7.972` | `7.979` | `82.79` | `82.72` | `+0.0069` | `RUS` |
| `Band 5 (3180-3920 m)` | `740` | `9.050` | `9.157` | `81.77` | `80.82` | `+0.1064` | `RUS` |
| `Band 6 (3920-4780 m)` | `860` | `10.888` | `11.018` | `78.98` | `78.05` | `+0.1297` | `RUS` |
| `Band 7 (4780-5823 m)` | `1043` | `16.942` | `16.994` | `61.56` | `61.37` | `+0.0527` | `RUS` |
| **Total / Sum** | **`5823`** | **`85.728`** | **`85.984`** | **`67.92`** | **`67.72`** | **`+0.2560`** | **`RUS`** |

* **Conservation & Bounds**: Every band $> 600\text{ m}$ averages between `50.38 m/s` and `82.79 m/s` (`[35, 90] m/s`). Band deltas sum to `+0.2560 s` with `0.000000 ms` conservation error.
