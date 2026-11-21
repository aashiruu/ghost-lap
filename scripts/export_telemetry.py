#!/usr/bin/env python3
import json
import os
import sys
from datetime import datetime, timedelta
import requests

DRIVERS = {
    "RUS": 63,
    "HAM": 44,
}

# Resolve 2024 Silverstone Qualifying session_key dynamically
sessions_url = "https://api.openf1.org/v1/sessions?year=2024&circuit_short_name=Silverstone&session_name=Qualifying"
sessions = requests.get(sessions_url, timeout=30).json()
if not sessions:
    print("Could not resolve 2024 Silverstone Qualifying session from OpenF1")
    sys.exit(1)

SESSION_KEY = str(sessions[0]["session_key"])
print(f"Resolved Session: {sessions[0].get('session_name')} | {sessions[0].get('date_start')} | session_key={SESSION_KEY}")

os.makedirs("data", exist_ok=True)

print("\n=== STEP 1: OPENF1 FASTEST Q3 LAP EXTRACTION & SANITY CHECK ===")
print(f"{'Driver':<6} | {'Lap':<4} | {'Samples':<7} | {'Hz':<5} | {'Start (UTC)':<24} | {'End (UTC)':<24} | {'Min km/h':<8} | {'Max km/h':<8} | {'Official (s)'}")
print("-" * 123)

for code, d_num in DRIVERS.items():
    laps_url = f"https://api.openf1.org/v1/laps?session_key={SESSION_KEY}&driver_number={d_num}"
    all_laps = requests.get(laps_url, timeout=30).json()
    valid_laps = [
        l for l in all_laps
        if l.get("lap_duration") is not None and not l.get("is_pit_out_lap", False)
    ]
    if not valid_laps:
        print(f"No valid flying laps found for {code}")
        sys.exit(1)

    fastest = min(valid_laps, key=lambda l: float(l["lap_duration"]))
    lap_num = fastest["lap_number"]
    lap_dur = float(fastest["lap_duration"])
    t_start = datetime.fromisoformat(fastest["date_start"].replace("Z", "+00:00"))
    t_end = t_start + timedelta(seconds=lap_dur)

    # Fetch with 0.5s buffer on both ends so boundary timestamps bracket [0, lap_dur]
    q_start = (t_start - timedelta(milliseconds=500)).strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"
    q_end = (t_end + timedelta(milliseconds=500)).strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"

    car_url = (
        f"https://api.openf1.org/v1/car_data?session_key={SESSION_KEY}&driver_number={d_num}"
        f"&date>={q_start}&date<={q_end}"
    )
    loc_url = (
        f"https://api.openf1.org/v1/location?session_key={SESSION_KEY}&driver_number={d_num}"
        f"&date>={q_start}&date<={q_end}"
    )

    car_res = sorted(requests.get(car_url, timeout=30).json(), key=lambda r: r["date"])
    loc_res = sorted(requests.get(loc_url, timeout=30).json(), key=lambda r: r["date"])

    if not car_res or not loc_res:
        print(f"Empty car_data or location response for {code}")
        sys.exit(1)

    loc_times = [datetime.fromisoformat(x["date"].replace("Z", "+00:00")).timestamp() for x in loc_res]

    samples = []
    cum_dist = 0.0
    prev_t = None

    for c in car_res:
        ct = datetime.fromisoformat(c["date"].replace("Z", "+00:00")).timestamp()
        rel_t = ct - t_start.timestamp()
        spd_kmh = float(c.get("speed", 0.0))

        if prev_t is not None:
            dt = ct - prev_t
            cum_dist += (spd_kmh / 3.6) * dt
        prev_t = ct

        best_idx = min(range(len(loc_times)), key=lambda i: abs(loc_times[i] - ct))
        if loc_times[best_idx] <= ct and best_idx < len(loc_res) - 1:
            i0, i1 = best_idx, best_idx + 1
        elif loc_times[best_idx] > ct and best_idx > 0:
            i0, i1 = best_idx - 1, best_idx
        else:
            i0 = i1 = best_idx

        t0, t1 = loc_times[i0], loc_times[i1]
        alpha = (ct - t0) / (t1 - t0) if t1 > t0 else 0.0
        alpha = max(0.0, min(1.0, alpha))

        x = float(loc_res[i0]["x"]) + alpha * (float(loc_res[i1]["x"]) - float(loc_res[i0]["x"]))
        y = float(loc_res[i0]["y"]) + alpha * (float(loc_res[i1]["y"]) - float(loc_res[i0]["y"]))
        z = float(loc_res[i0]["z"]) + alpha * (float(loc_res[i1]["z"]) - float(loc_res[i0]["z"]))

        samples.append({
            "rel_time_sec": round(rel_t, 4),
            "date": c["date"],
            "speed": spd_kmh,
            "throttle": float(c.get("throttle", 0.0)),
            "brake": int(c.get("brake", 0)) > 0,
            "gear": int(c.get("n_gear", 0)),
            "x": round(x, 3),
            "y": round(y, 3),
            "z": round(z, 3),
            "distance": round(cum_dist, 3),
        })

    speeds = [s["speed"] for s in samples]
    span_sec = samples[-1]["rel_time_sec"] - samples[0]["rel_time_sec"]
    hz = (len(samples) - 1) / span_sec if span_sec > 0 else 0.0

    print(
        f"{code:<6} | {lap_num:<4} | {len(samples):<7} | {hz:<5.2f} | "
        f"{samples[0]['date'][:23]:<24} | {samples[-1]['date'][:23]:<24} | "
        f"{min(speeds):<8.1f} | {max(speeds):<8.1f} | {lap_dur:.3f}"
    )

    out = {
        "driver_code": code,
        "session_key": SESSION_KEY,
        "lap_number": lap_num,
        "date_start": fastest["date_start"],
        "official_lap_time": lap_dur,
        "sample_count": len(samples),
        "samples": samples,
    }
    with open(f"data/{code}_lap.json", "w") as f:
        json.dump(out, f, indent=2)
