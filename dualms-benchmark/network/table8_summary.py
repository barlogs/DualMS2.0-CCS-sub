#!/usr/bin/env python3
"""Summarize coordinator JSONL output using the Table 8 metrics."""

import argparse
import json
import sys
from pathlib import Path


MEBIBYTE = 1024 * 1024


def input_file(value: str) -> Path:
    path = Path(value)
    if path.is_dir():
        path = path / "coordinator_summaries.jsonl"
    if not path.is_file():
        raise argparse.ArgumentTypeError(f"missing coordinator summary: {path}")
    return path


def mean(rows: list[dict], field: str) -> float:
    try:
        return sum(float(row[field]) for row in rows) / len(rows)
    except KeyError as error:
        raise ValueError(f"missing field {error.args[0]!r}") from error


def scheme_name(parameter_set: str) -> str:
    names = {
        "dualms-n8": "DualMS",
        "rounding01-n8": "DualMS2.0 (rounding)",
        "sec3-rounding01-n8": "DualMS2.0 (rounding)",
    }
    return names.get(parameter_set, parameter_set)


def load_summary(path: Path) -> dict:
    rows = []
    with path.open(encoding="utf-8") as handle:
        for line_number, line in enumerate(handle, start=1):
            if not line.strip():
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError as error:
                raise ValueError(f"{path}:{line_number}: invalid JSON: {error.msg}") from error
            if row.get("role") == "coordinator":
                rows.append(row)

    if not rows:
        raise ValueError(f"{path}: no coordinator summaries")

    parameter_sets = {str(row.get("parameter_set", "")) for row in rows}
    if len(parameter_sets) != 1:
        raise ValueError(f"{path}: mixed parameter sets: {sorted(parameter_sets)}")

    local_sign1 = mean(rows, "coordinator_sign1_success_ms")
    network_sign1 = mean(rows, "sign1_network_success_ms")
    local_sign2 = mean(rows, "coordinator_sign2_success_ms")
    network_sign2 = mean(rows, "sign2_network_success_ms")
    combine = mean(rows, "sagg_ms")

    return {
        "source": str(path),
        "runs": len(rows),
        "scheme": scheme_name(parameter_sets.pop()),
        "sign1_local": local_sign1,
        "sign1_network": network_sign1,
        "sign2_local": local_sign2,
        "sign2_network": network_sign2,
        "combine": combine,
        "t": mean(rows, "rejected_attempts"),
        "traffic_mib": sum(
            (float(row["bytes_in"]) + float(row["bytes_out"])) / MEBIBYTE
            for row in rows
        )
        / len(rows),
    }


def render_table(summaries: list[dict]) -> None:
    headers = [
        "Scheme",
        "Round",
        "Local Compute (ms)",
        "Network Latency (ms)",
        "Combine (ms)",
        "End-to-End (ms)",
        "t",
        "Traffic (MiB)",
    ]
    rows = []
    for summary in summaries:
        sign1_e2e = summary["sign1_local"] + summary["sign1_network"]
        sign2_e2e = summary["sign2_local"] + summary["sign2_network"] + summary["combine"]
        rows.extend(
            [
                [
                    summary["scheme"],
                    "Sign1",
                    f"{summary['sign1_local']:.3f}",
                    f"{summary['sign1_network']:.3f}",
                    "-",
                    f"{sign1_e2e:.3f}",
                    f"{summary['t']:.1f}",
                    f"{summary['traffic_mib']:.3f}",
                ],
                [
                    "",
                    "Sign2",
                    f"{summary['sign2_local']:.3f}",
                    f"{summary['sign2_network']:.3f}",
                    f"{summary['combine']:.3f}",
                    f"{sign2_e2e:.3f}",
                    "",
                    "",
                ],
            ]
        )

    widths = [len(header) for header in headers]
    for row in rows:
        for index, value in enumerate(row):
            widths[index] = max(widths[index], len(value))

    border = "+" + "+".join("-" * (width + 2) for width in widths) + "+"

    def print_row(row: list[str]) -> None:
        print("|" + "|".join(f" {value:<{widths[index]}} " for index, value in enumerate(row)) + "|")

    print("Table 8 WAN summary")
    for summary in summaries:
        print(f"{summary['scheme']}: {summary['runs']} run(s), source: {summary['source']}")
    print(border)
    print_row(headers)
    print(border)
    for row in rows:
        print_row(row)
    print(border)
    print("t is the mean rejected_attempts; Traffic is mean (bytes_in + bytes_out) / 2^20.")


def render_json(summaries: list[dict]) -> None:
    data = {"table": "Table 8", "schemes": []}
    for summary in summaries:
        sign1_e2e = summary["sign1_local"] + summary["sign1_network"]
        sign2_e2e = summary["sign2_local"] + summary["sign2_network"] + summary["combine"]
        data["schemes"].append(
            {
                "scheme": summary["scheme"],
                "runs": summary["runs"],
                "rounds": {
                    "sign1": {
                        "local_compute_ms": round(summary["sign1_local"], 3),
                        "network_latency_ms": round(summary["sign1_network"], 3),
                        "combine_ms": None,
                        "end_to_end_ms": round(sign1_e2e, 3),
                    },
                    "sign2": {
                        "local_compute_ms": round(summary["sign2_local"], 3),
                        "network_latency_ms": round(summary["sign2_network"], 3),
                        "combine_ms": round(summary["combine"], 3),
                        "end_to_end_ms": round(sign2_e2e, 3),
                    },
                },
                "mean_rejected_attempts": round(summary["t"], 1),
                "mean_traffic_mib": round(summary["traffic_mib"], 3),
            }
        )
    print(json.dumps(data, indent=2))


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Format coordinator JSONL output using the Table 8 WAN metrics."
    )
    parser.add_argument(
        "summaries",
        nargs="+",
        type=input_file,
        metavar="RUN_DIR_OR_JSONL",
        help="a network-runs directory or its coordinator_summaries.jsonl",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="print the Table 8 summary as JSON",
    )
    args = parser.parse_args()

    try:
        summaries = [load_summary(path) for path in args.summaries]
        if args.json:
            render_json(summaries)
        else:
            render_table(summaries)
    except ValueError as error:
        parser.error(str(error))
    return 0


if __name__ == "__main__":
    sys.exit(main())
