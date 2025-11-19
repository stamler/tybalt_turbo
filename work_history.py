#!/usr/bin/env -S uv run --with requests

# NOTE: Prompt caching typically requires a minimum of 1024 tokens (approx. 4000
# chars) for the static prefix. The current prompt is too short (~300 tokens) to
# trigger caching with most providers, so is not implemented.

# TODO: MULTITHREADING TO SPEED UP THE PROCESS, SENDING MULTIPLE REQUESTS IN PARALLEL
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
of exactly three values: "Tybalt", "Partial", or "IT". The script writes results
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
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests


# ========= CONFIGURE THESE VALUES =========

# Put your OpenRouter API key here.
OPENROUTER_API_KEY: str = "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"

# Choose your model here (see OpenRouter docs for available models).
# Examples: "openai/gpt-4.1-mini", "anthropic/claude-3.5-sonnet", etc.
OPENROUTER_MODEL: str = "x-ai/grok-4-fast"

# Optional: set a small delay between requests to be gentle on the API (in seconds).
REQUEST_DELAY_SECONDS: float = 0.2

# Maximum number of data rows to classify when running in --test mode.
TEST_MODE_MAX_ROWS: int = 32

# Number of concurrent worker threads to use for API requests.
# Set to 1 to force sequential processing.
NUM_REQUEST_THREADS: int = 16

# CSV column names for Partial breakdowns.
IT_DESCRIPTION_COLUMN = "IT_Description"
IT_HOURS_COLUMN = "IT_Hours"
TYBALT_DESCRIPTION_COLUMN = "Tybalt_Description"
TYBALT_HOURS_COLUMN = "Tybalt_Hours"

# =========================================

OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"

PROMPT_TEMPLATE = """You are a data preparation analyst assisting with the creation of a grant application
where we estimate the number of hours that were put into creating the Tybalt app, including programming.

Your task is to read a single work log entry and:
- classify it into exactly one of three categories: "Tybalt", "Partial", or "IT"; and
- when appropriate, decompose the logged hours between IT work and Tybalt work.

You will work according to the following framework:

- Return "Tybalt" if the work description is exclusively related to programming Tybalt or working on the 
  Tybalt app. This can include references to Tybalt, backend or frontend work, refactoring, component names
  (especially components starting with "DS"), or any other clearly programming-related task.
  If the work seems like it is not programming but it mentions Tybalt, then still treat it as "Tybalt". Meetings
  related to Tybalt are also Tybalt work.

- Return "IT" if the description contains nothing about programming Tybalt.
  In this dataset, all non-programming work is IT work unless it mentions Tybalt.

- Return "Partial" if some of the description is related to programming Tybalt but not all of it.
  For example, if "end user support" is included but the rest is programming-related, use "Partial".

You are given:
- workDescription: a free-form text description of the work
- total_hours: the total number of hours logged for this row

Your output MUST be a single JSON object, with no extra text, comments, or Markdown.

If the correct classification is "Tybalt" or "IT", you MUST return a JSON object with exactly one key:
- { "type": "Tybalt" }
  or
- { "type": "IT" }

If the correct classification is "Partial", you MUST return a JSON object with exactly these FIVE keys:
- "type": must be "Partial"
- "IT_Description": a short description of the IT portion of the work
- "IT_Hours": the number of hours (as a number) you believe were IT work
- "Tybalt_Description": a short description of the Tybalt-related portion of the work
- "Tybalt_Hours": the number of hours (as a number) you believe were Tybalt work

No other keys are allowed in the JSON.

For "Partial" classifications, the following constraint is CRITICAL:
- IT_Hours + Tybalt_Hours MUST equal total_hours exactly (within normal floating point rounding, e.g. within 0.01 hours).
  Think carefully about how you split the time and check your math before you respond.

A few guidelines for IT work:
- if the only IT portion of a workDescription is end user support is mentioned, then allocate just 1 hour for IT work.

Here is the work log entry:
- workDescription: %s
- total_hours: %s
"""

RETRY_PROMPT_TEMPLATE = """You previously returned JSON that did not satisfy the validation rules for this work log entry.

Work log entry:
- workDescription: %s
- total_hours: %s

Your previous JSON response was:
%s

Problem detected:
%s

Carefully reconsider the description and the rules, then respond AGAIN with a SINGLE corrected JSON object.
The rules are:
- "type" must be exactly one of "Tybalt", "Partial", or "IT".
- If "type" is "Tybalt" or "IT", the JSON object must contain ONLY the key "type".
- If "type" is "Partial", the JSON object must contain EXACTLY these keys:
  "type", "IT_Description", "IT_Hours", "Tybalt_Description", and "Tybalt_Hours".
- For "Partial", IT_Hours and Tybalt_Hours must be numeric values and IT_Hours + Tybalt_Hours must equal total_hours within 0.01 hours.

Reply with JSON ONLY. Do not include any explanation, prose, or Markdown.
"""


def build_prompt(work_description: str, total_hours: float) -> str:
    """Fill the prompt template with the given work description and hours."""
    return PROMPT_TEMPLATE % (work_description, total_hours)


def call_openrouter_messages(messages: List[Dict[str, str]]) -> str:
    """Call the OpenRouter chat completions API and return the model's raw text response."""
    if not OPENROUTER_API_KEY or OPENROUTER_API_KEY == "YOUR_OPENROUTER_API_KEY_HERE":
        raise RuntimeError("Please set OPENROUTER_API_KEY at the top of the script.")

    headers = {
        "Authorization": f"Bearer {OPENROUTER_API_KEY}",
        "Content-Type": "application/json",
    }

    body = {
        "model": OPENROUTER_MODEL,
        "messages": messages,
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
    Normalize the model output to exactly "Tybalt", "Partial", or "IT".

    This function is defensive: it trims whitespace/quotes and attempts a simple
    mapping based on keywords. If the output can't be mapped, it raises.
    """
    text = raw.strip().strip('"').strip("'")

    # Exact matches (case-insensitive)
    lowered = text.lower()
    if lowered == "tybalt":
        return "Tybalt"
    if lowered == "partial":
        return "Partial"
    if lowered == "it":
        return "IT"

    # Fuzzy mapping based on contained words
    if "tybalt" in lowered:
        return "Tybalt"
    if "partial" in lowered:
        return "Partial"
    if lowered in {"support", "it work", "it"}:
        return "IT"

    # If we get here, something went wrong with the instruction-following.
    raise ValueError(f"Could not map model output to a valid category: {raw!r}")


class ClassificationValidationError(Exception):
    """Base class for validation errors when interpreting model output."""


class HoursMismatchError(ClassificationValidationError):
    """Raised when IT_Hours + Tybalt_Hours does not match total_hours."""

    def __init__(self, it_hours: float, tybalt_hours: float, total_hours: float) -> None:
        super().__init__(
            f"IT_Hours ({it_hours}) + Tybalt_Hours ({tybalt_hours}) "
            f"does not equal total_hours ({total_hours})."
        )
        self.it_hours = it_hours
        self.tybalt_hours = tybalt_hours
        self.total_hours = total_hours


def parse_and_validate_response(raw: str, total_hours: float) -> Dict[str, Any]:
    """
    Parse the model's JSON response and validate it.

    Returns a normalized dict with:
      - always: "type" in {"Tybalt", "Partial", "IT"}
      - for "Partial": also
          IT_Description, IT_Hours (float),
          Tybalt_Description, Tybalt_Hours (float)

    Raises ClassificationValidationError if anything is invalid.
    """
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as e:
        raise ClassificationValidationError(f"Response is not valid JSON: {e}") from e

    if not isinstance(data, dict):
        raise ClassificationValidationError("Top-level JSON value must be an object.")

    if "type" not in data:
        raise ClassificationValidationError("JSON object is missing required key 'type'.")

    try:
        category = normalize_category(str(data["type"]))
    except ValueError as e:
        raise ClassificationValidationError(str(e)) from e

    # Non-partial: only "type" allowed.
    if category in {"Tybalt", "IT"}:
        extra_keys = [k for k in data.keys() if k != "type"]
        if extra_keys:
            raise ClassificationValidationError(
                f"Unexpected extra keys for non-partial classification {category!r}: {extra_keys}"
            )
        return {"type": category}

    # Partial: require exactly the five keys.
    required_keys = {
        "type",
        IT_DESCRIPTION_COLUMN,
        IT_HOURS_COLUMN,
        TYBALT_DESCRIPTION_COLUMN,
        TYBALT_HOURS_COLUMN,
    }
    missing = [k for k in required_keys if k not in data]
    if missing:
        raise ClassificationValidationError(
            f"JSON object for 'Partial' is missing required keys: {missing}"
        )

    extra = [k for k in data.keys() if k not in required_keys]
    if extra:
        raise ClassificationValidationError(
            f"JSON object for 'Partial' includes unexpected keys: {extra}"
        )

    # Validate hours and sum.
    try:
        it_hours = float(data[IT_HOURS_COLUMN])
        tybalt_hours = float(data[TYBALT_HOURS_COLUMN])
    except (TypeError, ValueError) as e:
        raise ClassificationValidationError(
            f"{IT_HOURS_COLUMN} and {TYBALT_HOURS_COLUMN} must be numeric."
        ) from e

    if it_hours < 0 or tybalt_hours < 0:
        raise ClassificationValidationError(
            f"{IT_HOURS_COLUMN} and {TYBALT_HOURS_COLUMN} must be non-negative."
        )

    if total_hours < 0:
        raise ClassificationValidationError("total_hours must be non-negative.")

    # Allow a small tolerance for floating point rounding.
    if abs((it_hours + tybalt_hours) - total_hours) > 0.01:
        raise HoursMismatchError(it_hours, tybalt_hours, total_hours)

    return {
        "type": category,
        IT_DESCRIPTION_COLUMN: str(data[IT_DESCRIPTION_COLUMN]).strip(),
        IT_HOURS_COLUMN: it_hours,
        TYBALT_DESCRIPTION_COLUMN: str(data[TYBALT_DESCRIPTION_COLUMN]).strip(),
        TYBALT_HOURS_COLUMN: tybalt_hours,
    }


def build_retry_prompt(
    work_description: str,
    total_hours: float,
    previous_json: str,
    problem: str,
) -> str:
    """Fill the retry prompt template with context and the specific validation problem."""
    return RETRY_PROMPT_TEMPLATE % (
        work_description,
        total_hours,
        previous_json,
        problem,
    )


def classify_entry(work_description: str, total_hours: float) -> Dict[str, Any]:
    """
    Classify a single work description (with total_hours) and return a normalized dict.

    The dict always contains "type" in {"Tybalt", "Partial", "IT"} and,
    for "Partial", also includes the IT/Tybalt breakdown fields.
    """
    prompt = build_prompt(work_description, total_hours)
    # First attempt.
    raw_response = call_openrouter_messages(
        [
            {
                "role": "user",
                "content": prompt,
            }
        ]
    )

    try:
        result = parse_and_validate_response(raw_response, total_hours)
    except ClassificationValidationError as e:
        # Retry once with explicit feedback, including the original "conversation"
        # (prompt + previous JSON) and a clear message about what went wrong.
        retry_prompt = build_retry_prompt(
            work_description=work_description,
            total_hours=total_hours,
            previous_json=raw_response,
            problem=str(e),
        )
        raw_response_retry = call_openrouter_messages(
            [
                {"role": "user", "content": prompt},
                {"role": "assistant", "content": raw_response},
                {"role": "user", "content": retry_prompt},
            ]
        )
        result = parse_and_validate_response(raw_response_retry, total_hours)

    if REQUEST_DELAY_SECONDS > 0:
        time.sleep(REQUEST_DELAY_SECONDS)

    return result


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
    start_time = time.monotonic()

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

    # Ensure the Partial breakdown columns exist in the CSV header.
    for col in (
        IT_DESCRIPTION_COLUMN,
        IT_HOURS_COLUMN,
        TYBALT_DESCRIPTION_COLUMN,
        TYBALT_HOURS_COLUMN,
    ):
        if col not in fieldnames:
            fieldnames.append(col)

    total = len(rows)
    print(f"Found {total} rows.")

    if test_mode:
        print(
            f"Test mode enabled: only the first {TEST_MODE_MAX_ROWS} data "
            "rows will be classified."
        )

    # Collect rows that actually need classification.
    to_classify = []  # list of (idx, row, desc, total_hours)

    for idx, row in enumerate(rows, start=1):
        # In test mode, stop completely after examining the first N rows.
        if test_mode and idx > TEST_MODE_MAX_ROWS:
            break

        desc = (row.get("workDescription") or "").strip()
        existing_category: Optional[str] = (row.get("category") or "").strip()

        # Optional: skip rows that already have a valid category
        if existing_category in {"Tybalt", "Partial", "IT"}:
            print(f"[{idx}/{total}] Skipping (already categorized as {existing_category}).")
            continue

        # Parse total hours for this row. Prefer "hours", fall back to "jobHours" if present.
        hours_str = (
            (row.get("hours") or row.get("Hours") or row.get("jobHours") or "").strip()
        )
        if not hours_str:
            print(
                f"[{idx}/{total}] Missing hours value; skipping classification for this row.",
                file=sys.stderr,
            )
            continue
        try:
            total_hours_for_row = float(hours_str)
        except ValueError:
            print(
                f"[{idx}/{total}] Invalid hours value {hours_str!r}; skipping classification for this row.",
                file=sys.stderr,
            )
            continue

        if not desc:
            print(f"[{idx}/{total}] Empty description; categorizing as IT.")
            row["category"] = "IT"
            # Ensure breakdown columns are empty for non-partial rows.
            row[IT_DESCRIPTION_COLUMN] = ""
            row[IT_HOURS_COLUMN] = ""
            row[TYBALT_DESCRIPTION_COLUMN] = ""
            row[TYBALT_HOURS_COLUMN] = ""
            continue

        print(f"[{idx}/{total}] Queuing description for classification...")
        to_classify.append((idx, row, desc, total_hours_for_row))

    if to_classify:
        max_workers = max(1, NUM_REQUEST_THREADS)
        print(f"Dispatching {len(to_classify)} classification request(s) with {max_workers} worker thread(s).")

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_meta = {
                executor.submit(classify_entry, desc, total_hours_for_row): (idx, row)
                for idx, row, desc, total_hours_for_row in to_classify
            }

            for future in as_completed(future_to_meta):
                idx, row = future_to_meta[future]
                try:
                    result = future.result()
                    category = result["type"]
                    row["category"] = category

                    if category == "Partial":
                        row[IT_DESCRIPTION_COLUMN] = result.get(IT_DESCRIPTION_COLUMN, "")
                        row[IT_HOURS_COLUMN] = result.get(IT_HOURS_COLUMN, "")
                        row[TYBALT_DESCRIPTION_COLUMN] = result.get(TYBALT_DESCRIPTION_COLUMN, "")
                        row[TYBALT_HOURS_COLUMN] = result.get(TYBALT_HOURS_COLUMN, "")
                    else:
                        # Ensure breakdown columns are empty for non-partial rows.
                        row[IT_DESCRIPTION_COLUMN] = ""
                        row[IT_HOURS_COLUMN] = ""
                        row[TYBALT_DESCRIPTION_COLUMN] = ""
                        row[TYBALT_HOURS_COLUMN] = ""

                    print(f"[{idx}/{total}] -> {category}")
                except Exception as e:
                    # Log the error but do NOT overwrite category; leave it blank or whatever it was.
                    print(f"    ERROR while classifying row {idx}: {e}", file=sys.stderr)

    output_path = input_path.with_name(f"{input_path.stem}_output{input_path.suffix}")
    print(f"Writing updated CSV to: {output_path}")
    write_csv(output_path, rows, fieldnames)
    print("Done.")

    elapsed = time.monotonic() - start_time
    print(f"Total runtime: {elapsed:.2f} seconds.")

    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))