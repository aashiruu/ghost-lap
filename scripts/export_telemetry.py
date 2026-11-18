#!/usr/bin/env python3
import json
import os
import sys
from datetime import datetime, timedelta
import requests

SESSION_KEY = "9558"  # 2024 Silverstone Qualifying
DRIVERS = {
    "RUS": 63,
    "HAM": 44,
}

os.makedirs("data", exist_ok=True)

for code, d_num in DRIVERS.items():
    laps_url = f"https://api.openf1.org/v1/laps?session_key={SESSION_KEY}&driver_number={d_num}"
    laps = [l for l in requests.get(laps_url).json() if l.get("lap_duration") and not l.get("is_pit_out_lap")]
    fastest = min(laps, key=lambda l: l["lap_duration"])

    t_start = datetime.fromisoformat(fastest["date_start"].replace("Z", "+00:00"))
    lap_dur = float(fastest["lap_duration"])
    t_end = t_start + timedelta(seconds=lap_dur)

    iso_start = t_start.strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"
    iso_end = t_end.strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"

    car_url = (
        f"https://api.openf1.org/v1/car_data?session_key={SESSION_KEY}&driver_number={d_num}"
        f"&date>={iso_start}&date<={iso_end}"
    )
    loc_url = (
        f"https://api.openf1.org/v1/location?session_key={SESSION_KEY}&driver_number={d_num}"
        f"&date>={iso_start}&date<={iso_end}"
    )
    car_res = requests.get(car_url).json()
    loc_res = requests.get(loc_url).json()

    if not car_res or not loc_res:
        print(f"Failed to fetch lap window for {code}")
        sys.exit(1)

    # Nearest-timestamp merge between ~3.7 Hz car_data and ~3.7 Hz location
    loc_times = [datetime.fromisoformat(x["date"].replace("Z", "+00:00")).timestamp() for x in loc_res]
    samples = []
    cum_dist = 0.0
    prev_t = None

    for c in car_res:
        ct = datetime.fromisoformat(c["date"].replace("Z", "+00:00")).timestamp()
        rel_t = max(0.0, ct - t_start.timestamp())
        if prev_t is not None:
            dt = ct - prev_t
            cum_dist += (float(c.get("speed", 0)) / 3.6) * dt
        prev_t = ct

        # Closest location frame
        best_idx = min(range(len(loc_times)), key=lambda i: abs(loc_times[i] - ct))
        loc = loc_res[best_idx]

        samples.append({
            "rel_time_sec": round(rel_t, 4),
            "date": c["date"],
            "speed": float(c.get("speed", 0)),
            "throttle": float(c.get("throttle", 0)),
            "brake": int(c.get("brake", 0)) > 0,
            "gear": int(c.get("n_gear", 0)),
            "x": float(loc.get("x", 0)),
            "y": float(loc.get("y", 0)),
            "z": float(loc.get("z", 0)),
            "distance": round(cum_dist, 3),
        })

    out = {
        "official_lap_time": lap_dur,
        "sample_count": len(samples),
        "samples": samples,
    }
    with open(f"data/{code}_lap.json", "w") as f:
        json.dump(out, f, indent=2)
    print(f"Exported {code}: {len(samples)} samples over {lap_dur:.3f}s")
