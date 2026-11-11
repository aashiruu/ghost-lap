# Architectural Trade-Offs and Decisions

## Stage 0: Telemetry Ingestion & Serialization
* **Decision**: Offline JSON export via FastF1 wrapper instead of direct live timing stream.
* **Trade-Off**: Eliminates network failure noise and remote rate-limiting during local pipeline testing at the cost of requiring a local Python pre-processing step.
* **Data Granularity**: FastF1 merges channels into ~8 Hz non-uniform streams. We accept non-uniform sampling intervals in the raw stream and defer uniform grid resampling to Stage 3.
## Stage 1: Replay Producer Strategy
* **Decision**: Single topic (`telemetry-raw`) with driver code as partition key (`RUS`, `HAM`) and time-delta ticker delay scaling.
* **Trade-Off**: Pushing merged driver telemetry over local Redpanda socket introduces ~1-2ms per-sample network overhead compared to pure memory slices, but guarantees isolated partition consumption and realistic streaming event semantics.
* **Replay Control**: Configurable multiplier (`-speed=1.0`, `-speed=5.0`) scales target sleep delays accurately down to sub-millisecond precision.
