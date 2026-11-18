# Architectural Trade-Offs and Decisions

## Stage 0: Telemetry Ingestion and Serialization
* **Decision**: Offline JSON export via FastF1 wrapper instead of direct live timing stream.
* **Trade-Off**: Eliminates network failure noise and remote rate-limiting during local pipeline testing at the cost of requiring a local Python pre-processing step.
* **Data Granularity**: FastF1 merges channels into ~8 Hz non-uniform streams. We accept non-uniform sampling intervals in the raw stream and defer uniform grid resampling to Stage 3.
## Stage 1: Replay Producer Strategy
* **Decision**: Single topic (`telemetry-raw`) with driver code as partition key (`RUS`, `HAM`) and time-delta ticker delay scaling.
* **Trade-Off**: Pushing merged driver telemetry over local Redpanda socket introduces ~1-2ms per-sample network overhead compared to pure memory slices, but guarantees isolated partition consumption and realistic streaming event semantics.
* **Replay Control**: Configurable multiplier (`-speed=1.0`, `-speed=5.0`) scales target sleep delays accurately down to sub-millisecond precision.
## Stage 2: Track Distance Alignment Trade-Offs
* **Integration ($\int v \, dt$) vs Spatial Projection ($X,Y \to S$)**:
  * **Integration**: Incurs numerical integration drift across 85+ seconds of lap time. Speed sample interval variations trigger compound distance offset over the lap.
  * **Spatial Projection**: Locks spatial distance $S$ to track coordinates, avoiding time-integration drift, but depends on reference track resolution and spatial lookup precision.
## Stage 3: Distance Grid Resampling and Interpolation
* **Linear vs Spline Interpolation**:
  * **Linear Interpolation**: Adopted for hot-path streaming resampling. It enforces strict monotonicity, eliminates overshoot risk on step functions (brake on/off, throttle snaps), and runs in $O(1)$ bounded time per distance step.
  * **Cubic Spline**: Higher continuity for smooth curves, but risks ringing/overshoot on binary signal transitions without heavy clamping guardrails.
## Stage 4: Delta and Ground-Truth Verification
* **Finish Line Delta Match**: Time delta calculated at the final grid distance point ($S_{\text{finish}}$) must match official timing difference ($+0.171\text{s}$) within strict numerical bounds ($\le 50\text{ms}$).

## Stage 5: Streaming Consumer Architecture
* **State Ingestion & Incremental Resampling**:
  * Telemetry frames are pushed into per-driver state buffers (`DriverStreamState`).
  * As new spatial distance milestones are crossed, resampled 1m grid points are computed incrementally without re-processing previous frames.
## Stage 6: Sample Loss Degradation & Resilience
* **Reproducible Seeded Random Dropper**:
  * Evaluated resilience against 5%, 10%, and 25% packet loss using PRNG seed `42`.
  * Preserved lap boundary endpoints to test linear interpolation gap-bridging fidelity under non-uniform frame drops.
## Stage 7: Feature-Based Track Segmentation
* **Decision**: Partitioned the 5,856 m grid into 7 contiguous track segments rather than 3 coarse FIA sectors.
* **Trade-Off**: Isolates corner-complex delta swings while enforcing boundary continuity ($S_{j,\text{end}} = S_{j+1,\text{start}}$) so segment deltas conserve the total lap gap (`0.000000 ms` conservation error).
