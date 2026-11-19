#!/usr/bin/env python3
import json
import math
import os

os.makedirs("data", exist_ok=True)

def build_lap(lap_time_sec: float, speed_offset: float):
    sample_count = 690
    samples = []
    cum_dist = 0.0
    dt = lap_time_sec / (sample_count - 1)

    for i in range(sample_count):
        t = i * dt
        theta = (i / (sample_count - 1)) * 2.0 * math.pi
        # Realistic oscillating corner/straight speed profile (110 to 315 km/h)
        speed = 215.0 + 95.0 * math.sin(3.0 * theta) + speed_offset * math.cos(2.0 * theta)
        throttle = 100.0 if math.sin(3.0 * theta) > -0.2 else 0.0
        brake = math.sin(3.0 * theta) < -0.55
        gear = max(2, min(8, int(speed // 38)))

        if i > 0:
            cum_dist += (speed / 3.6) * dt

        x = 400.0 * math.cos(theta) + 120.0 * math.sin(2.0 * theta)
        y = 260.0 * math.sin(theta) + 70.0 * math.cos(3.0 * theta)

        samples.append({
            "rel_time_sec": round(t, 4),
            "date": f"2024-07-06T15:55:{int(t % 60):02d}Z",
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
        "official_lap_time": lap_time_sec,
        "sample_count": sample_count,
        "samples": samples,
    }

with open("data/RUS_lap.json", "w") as f:
    json.dump(build_lap(85.819, 2.5), f)

with open("data/HAM_lap.json", "w") as f:
    json.dump(build_lap(85.990, -2.5), f)

print("Generated CI telemetry fixtures (RUS 85.819s, HAM 85.990s).")
