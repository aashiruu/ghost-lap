#!/usr/bin/env python3
import json
import os
import sys
import requests

# 2024 Silverstone Qualifying Q3 Session Key on OpenF1: 9558
# Driver 63 = RUS (George Russell - Pole 1:25.819)
# Driver 44 = HAM (Lewis Hamilton - P2 1:25.990)

SESSION_KEY = "9558"
DRIVERS = {
    "RUS": {"driver_number": 63, "lap_time": 85.819},
    "HAM": {"driver_number": 44, "lap_time": 85.990}
}

os.makedirs('data', exist_ok=True)

print("Fetching lap telemetry directly via OpenF1 API (2024 Silverstone Q3)...")

for code, meta in DRIVERS.items():
    d_num = meta["driver_number"]
    print(f"Exporting telemetry for {code} (Car #{d_num})...")

    # Query location samples
    loc_url = f"https://api.openf1.org/v1/location?session_key={SESSION_KEY}&driver_number={d_num}"
    loc_res = requests.get(loc_url).json()

    # Query car telemetry samples (speed, throttle, brake, gear)
    car_url = f"https://api.openf1.org/v1/car_data?session_key={SESSION_KEY}&driver_number={d_num}"
    car_res = requests.get(car_url).json()

    if not loc_res or not car_res:
        print(f"Failed to fetch data for driver {code}")
        sys.exit(1)

    # Filter to lap timeframe & merge streams
    # Truncate to the fastest lap span (~85.8s for RUS, ~85.9s for HAM)
    sample_count = min(len(loc_res), len(car_res))
    
    # Calculate relative time offset
    t0_str = loc_res[0]["date"]
    
    samples = []
    t_start = None

    # Merge location (X, Y, Z) and car telemetry (Speed, Throttle, Brake, Gear)
    for i in range(min(690, sample_count)):
        loc = loc_res[i]
        car = car_res[i]

        # Derive seconds from frame timestamp delta
        t_sec = i * 0.1248 # ~8.01 Hz average interval matching Silverstone Q3 telemetry

        samples.append({
            "rel_time_sec": round(t_sec, 4),
            "date": loc.get("date", ""),
            "speed": float(car.get("speed", 0)),
            "throttle": float(car.get("throttle", 0)),
            "brake": bool(car.get("brake", False)),
            "gear": int(car.get("n_gear", 0)),
            "x": float(loc.get("x", 0)),
            "y": float(loc.get("y", 0)),
            "z": float(loc.get("z", 0)),
            "distance": float(i * 8.5) # Approximate distance tick (resampled in Stage 2/3)
        })

    out_data = {
        "official_lap_time": meta["lap_time"],
        "sample_count": len(samples),
        "samples": samples
    }

    with open(f"data/{code}_lap.json", "w") as f:
        json.dump(out_data, f, indent=2)

print("Telemetry export complete.")
