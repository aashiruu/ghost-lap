#!/usr/bin/env python3
import json
import math
import os

os.makedirs("data", exist_ok=True)

def build_lap(code: str, lap_num: int, lap_time_sec: float, speed_offset: float):
    # ~3.8 Hz with 0.5s buffer on both ends so TrimToOfficialLapWindow brackets [0, lap_time_sec]
    sample_count = 330
    samples = []
    cum_dist = 0.0
    t_start = -0.35
    t_end = lap_time_sec + 0.35
    dt = (t_end - t_start) / (sample_count - 1)

    # Perimeter ~ 5825 m (in decimeters: 58250 units)
    rx, ry = 10800.0, 7400.0

    for i in range(sample_count):
        t = t_start + i * dt
        theta = max(0.0, min(1.0, t / lap_time_sec)) * 2.0 * math.pi
        # Realistic F1 Silverstone speed profile: ~135 km/h slow corners to ~320 km/h straights
        speed = 235.0 + 75.0 * math.sin(3.0 * theta) + speed_offset * math.cos(2.0 * theta)
        throttle = 100.0 if math.sin(3.0 * theta) > -0.25 else 0.0
        brake = math.sin(3.0 * theta) < -0.55
        gear = max(3, min(8, int(speed // 38)))

        if i > 0:
            cum_dist += (speed / 3.6) * dt

        x = rx * math.cos(theta) + 1400.0 * math.sin(2.0 * theta)
        y = ry * math.sin(theta) + 900.0 * math.cos(3.0 * theta)

        samples.append({
            "rel_time_sec": round(t, 4),
            "date": f"2024-07-06T15:08:{int(max(0, t) % 60):02d}Z",
            "speed": round(speed, 2),
            "throttle": throttle,
            "brake": brake,
            "gear": gear,
            "x": round(x, 3),
            "y": round(y, 3),
            "z": 0.0,
            "distance": round(cum_dist, 3),
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

print("Generated deterministic Silverstone Q3 CI fixtures (RUS 85.819s, HAM 85.990s).")
