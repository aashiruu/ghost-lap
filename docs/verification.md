# Verification Records and Execution Log

## Stage 0 Setup Verification
* Verified presence of exported JSON datasets in `data/RUS_lap.json` and `data/HAM_lap.json`.
* Sample counts verified: RUS = 688 samples, HAM = 689 samples.
* Average sampling rate verified: ~8.01 Hz.
## Stage 1 Producer Verification Log
* verified local Redpanda topic creation (`telemetry-raw`).
* Verified sample delivery order across drivers sorted by `rel_time_sec`.
* Verified speed scaling: 1380 total samples published to Redpanda in ~8.5s at 10x replay speed (`-speed=10.0`).
