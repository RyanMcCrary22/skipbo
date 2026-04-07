import os
from pathlib import Path

checkpoints_dir = Path("/Users/ryanmccrary/FunStuff/skipbo/runs/2026-04-04_09-33-50/checkpoints")

keep_steps = {
    # 1M cluster
    950000, 1000000, 1050000,
    # 10M cluster
    9950000, 10000000, 10050000,
    # 20M cluster
    19950000, 20000000, 20050000,
    # 28.35M cluster
    28250000, 28300000, 28350000
}

files_deleted = 0
for file in checkpoints_dir.iterdir():
    if not file.is_file():
        continue
    if file.suffix not in ['.zip']:
        continue
        
    try:
        # e.g. eval_1000000.json or model_1000000.zip
        stem = file.stem
        # parse step number
        parts = stem.split("_")
        if len(parts) == 2 and parts[1].isdigit():
            step = int(parts[1])
            if step not in keep_steps:
                file.unlink()
                files_deleted += 1
    except Exception as e:
        print(f"Error processing {file.name}: {e}")

print(f"Cleanup complete. Deleted {files_deleted} .zip checkpoint files.")
