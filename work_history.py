#!/usr/bin/env -S uv run --with requests
"""
Work History Categorizer, which processes the results of the following MySQL query:

SELECT date, hours, jobHours, workDescription FROM TimeEntries 
WHERE uid = 'UAmV8K6DcXVhSrMAtZua0OmCUPu2'
AND timetype = 'R'
AND workDescription IS NOT NULL
AND date >= '2023-11-01'
ORDER BY date DESC

This script reads a CSV work history file (e.g. "workHistory.csv"), sends each row's
"workDescription" text to an OpenRouter LLM with a classification prompt, and writes
the model's response into a new "category" column for that row. The category is one
of exactly three values: "Programming", "Partial", or "IT". The script writes results
to a new CSV file named "<input_stem>_output.csv" (for example, "workHistory_output.csv"),
preserving the original file. It supports an optional "--test" flag to classify only
the first 10 data rows (useful for smoke‑testing prompt/model behavior).

It can be run as:

    # Using uv via the shebang (after chmod +x):
    ./work_history.py workHistory.csv
    ./work_history.py --test workHistory.csv

    # Or explicitly with uv:
    uv run --with requests work_history.py workHistory.csv
    uv run --with requests work_history.py --test workHistory.csv

    # Or with a plain Python interpreter (requires `requests` to be installed):
    python3 work_history.py workHistory.csv

You must set your OpenRouter API key and model name in the constants below.
"""

import csv
import json
import sys
import time
from pathlib import Path
from typing import List, Dict, Any, Optional

import requests


# ========= CONFIGURE THESE VALUES =========

# Put your OpenRouter API key here.
OPENROUTER_API_KEY: str = "YOUR_OPENROUTER_API_KEY_HERE"

# Choose your model here (see OpenRouter docs for available models).
# Examples: "openai/gpt-4.1-mini", "anthropic/claude-3.5-sonnet", etc.
OPENROUTER_MODEL: str = "openai/gpt-4.1-mini"

# Optional: set a small delay between requests to be gentle on the API (in seconds).
REQUEST_DELAY_SECONDS: float = 0.2

# =========================================

OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"

PROMPT_TEMPLATE = """You are a data preparation analyst assisting with the creation of a grant application where we estimate the number of hours that were put into programming the tybalt app.

Your task is to categorize work descriptions into one of three categories: "Programming", "Partial", and "IT".
You must only return one of these 3 values and nothing else.

You will work according to the following framework:

- Return "Programming" if the work description is exclusively related to programming tybalt.
  This can include references to tybalt, backend or frontend work, refactoring, component names
  (especially components starting with "DS"), or any other clearly programming-related task.

- Return "IT" if the description contains nothing about programming tybalt. In this dataset,
  all non-programming work is IT work.

- Return "Partial" if some of the description is related to programming tybalt but not all of it.
  For example, if "end user support" is included but the rest is programming-related, use "Partial".

Remember: only return exactly one of "Programming", "Partial", or "IT" with no extra text.

HERE IS THE WORK DESCRIPTION:

%s
"""


def build_prompt(work_description: str) -> str:
    """Fill the prompt template with the given work description."""
    return PROMPT_TEMPLATE % work_description


def call_openrouter(prompt: str) -> str:
    """Call the OpenRouter chat completions API and return the model's raw text response."""
    if not OPENROUTER_API_KEY or OPENROUTER_API_KEY == "YOUR_OPENROUTER_API_KEY_HERE":
        raise RuntimeError("Please set OPENROUTER_API_KEY at the top of the script.")

    headers = {
        "Authorization": f"Bearer {OPENROUTER_API_KEY}",
        "Content-Type": "application/json",
    }

    body = {
        "model": OPENROUTER_MODEL,
        "messages": [
            {
                "role": "user",
                "content": prompt,
            }
        ],
        "temperature": 0.0,
    }

    resp = requests.post(OPENROUTER_URL, headers=headers, data=json.dumps(body), timeout=60)
    resp.raise_for_status()
    data = resp.json()

    try:
        content = data["choices"][0]["message"]["content"]
    except (KeyError, IndexError) as e:
        raise RuntimeError(f"Unexpected OpenRouter response format: {data}") from e

    return content.strip()


def normalize_category(raw: str) -> str:
    """
    Normalize the model output to exactly "Programming", "Partial", or "IT".

    This function is defensive: it trims whitespace/quotes and attempts a simple
    mapping based on keywords. If the output can't be mapped, it raises.
    """
    text = raw.strip().strip('"').strip("'")

    # Exact matches (case-insensitive)
    lowered = text.lower()
    if lowered == "programming":
        return "Programming"
    if lowered == "partial":
        return "Partial"
    if lowered == "it":
        return "IT"

    # Fuzzy mapping based on contained words
    if "programming" in lowered:
        return "Programming"
    if "partial" in lowered:
        return "Partial"
    if lowered in {"support", "it work", "it"}:
        return "IT"

    # If we get here, something went wrong with the instruction-following.
    raise ValueError(f"Could not map model output to a valid category: {raw!r}")


def read_csv(path: Path) -> (List[Dict[str, Any]], List[str]):
    """Read the CSV into a list of dicts and return (rows, fieldnames)."""
    with path.open("r", newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        rows = list(reader)
        fieldnames = reader.fieldnames or []
    return rows, fieldnames


def write_csv(path: Path, rows: List[Dict[str, Any]], fieldnames: List[str]) -> None:
    """Write rows to CSV using the given fieldnames."""
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)


def main(argv: List[str]) -> int:
    if len(argv) not in (2, 3):
        print("Usage: work_history.py [--test] workHistory.csv", file=sys.stderr)
        return 1

    test_mode = False
    args = argv[1:]

    if args[0] == "--test":
        test_mode = True
        if len(args) != 2:
            print("Usage: work_history.py [--test] workHistory.csv", file=sys.stderr)
            return 1
        csv_arg = args[1]
    else:
        if len(args) == 2 and args[1] == "--test":
            test_mode = True
        csv_arg = args[0]

    input_path = Path(csv_arg)
    if not input_path.exists():
        print(f"Error: file not found: {input_path}", file=sys.stderr)
        return 1

    print(f"Reading CSV from: {input_path}")
    rows, fieldnames = read_csv(input_path)

    if "workDescription" not in fieldnames:
        print('Error: CSV is missing a "workDescription" column.', file=sys.stderr)
        return 1

    if "category" not in fieldnames:
        fieldnames.append("category")

    total = len(rows)
    print(f"Found {total} rows.")

    if test_mode:
        print("Test mode enabled: only the first 10 data rows will be classified.")

    for idx, row in enumerate(rows, start=1):
        # In test mode, stop completely after examining the first 10 rows.
        if test_mode and idx > 10:
            break

        desc = (row.get("workDescription") or "").strip()
        existing_category: Optional[str] = (row.get("category") or "").strip()

        # Optional: skip rows that already have a valid category
        if existing_category in {"Programming", "Partial", "IT"}:
            print(f"[{idx}/{total}] Skipping (already categorized as {existing_category}).")
            continue

        if not desc:
            print(f"[{idx}/{total}] Empty description; categorizing as IT.")
            row["category"] = "IT"
            continue

        print(f"[{idx}/{total}] Classifying description...")
        prompt = build_prompt(desc)

        try:
            raw_response = call_openrouter(prompt)
            category = normalize_category(raw_response)
            row["category"] = category
            print(f"    -> {category}")
        except Exception as e:
            # Log the error but do NOT overwrite category; leave it blank or whatever it was.
            print(f"    ERROR while classifying row {idx}: {e}", file=sys.stderr)

        if REQUEST_DELAY_SECONDS > 0:
            time.sleep(REQUEST_DELAY_SECONDS)

    output_path = input_path.with_name(f"{input_path.stem}_output{input_path.suffix}")
    print(f"Writing updated CSV to: {output_path}")
    write_csv(output_path, rows, fieldnames)
    print("Done.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))