#!/usr/bin/env python3
import json
import math
import os

os.makedirs("data", exist_ok=True)

def build_lap(code: str, lap_num: int, lap_time_sec: float, speed_offset: float):
    sample_count = 330
    t_start = -0.35
    t_end = lap_time_sec + 0.35
    dt = (t_end - t_start) / (sample_count - 1)

    raw_rows = []
    cum_dist = 0.0
    lap_start_dist = 0.0
    lap_end_dist = 0.0

    for i in range(sample_count):
        t = t_start + i * dt
        phase = max(0.0, min(1.0, t / lap_time_sec)) * 2.0 * math.pi
        # Realistic Silverstone corner/straight speed profile: ~185 km/h (51 m/s) to ~305 km/h (85 m/s)
        speed = 244.0 + 58.0 * math.sin(3.0 * phase) + speed_offset * math.cos(2.0 * phase)
        if i > 0:
            prev_speed = raw_rows[-1]["speed"]
            cum_dist += (((prev_speed + speed) / 2.0) / 3.6) * dt
        if t <= 0.0:
            lap_start_dist = cum_dist
        if t <= lap_time_sec:
            lap_end_dist = cum_dist

        raw_rows.append({
            "rel_time_sec": round(t, 4),
            "date": f"2024-07-06T15:08:{int(max(0, t) % 60):02d}Z",
            "speed": round(speed, 2),
            "throttle": 100.0 if math.sin(3.0 * phase) > -0.25 else 0.0,
            "brake": math.sin(3.0 * phase) < -0.55,
            "gear": max(3, min(8, int(speed // 38))),
            "cum_dist": cum_dist,
        })

    # Place (x, y) on a circle of matching perimeter (in decimeters) so arc-length matches speed distance
    lap_len_m = max(1.0, lap_end_dist - lap_start_dist)
    radius_dm = (lap_len_m * 10.0) / (2.0 * math.pi)

    samples = []
    for r in raw_rows:
        theta = ((r["cum_dist"] - lap_start_dist) / lap_len_m) * 2.0 * math.pi
        samples.append({
            "rel_time_sec": r["rel_time_sec"],
            "date": r["date"],
            "speed": r["speed"],
            "throttle": r["throttle"],
            "brake": r["brake"],
            "gear": r["gear"],
            "x": round(radius_dm * math.cos(theta), 3),
            "y": round(radius_dm * math.sin(theta), 3),
            "z": 0.0,
            "distance": round(r["cum_dist"], 3),
        })

    return {
        "driver_code": code,
        "session_key": "9554",
        "lap_number": lap_num,
        "date_start": "2024-07-06T15:08:48.644Z",
        "official_lap_time": lap_time_sec,
        "sample_count": sample_count,
        "samples": samples,
    }

with open("data/RUS_lap.json", "w") as f:
    json.dump(build_lap("RUS", 25, 85.819, 2.5), f)

with open("data/HAM_lap.json", "w") as f:
    json.dump(build_lap("HAM", 24, 85.990, -2.5), f)

print("Generated physically consistent Silverstone Q3 CI fixtures (RUS 85.819s, HAM 85.990s).")
