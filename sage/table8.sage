import json
import os
import sys

try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)


DEFAULT_SUMMARIES = [
    os.path.abspath(os.path.join(
        _HERE,
        "..",
        "dualms-benchmark",
        "network",
        "wan-performance-table8",
        "table8-summary.json",
    )),
    os.path.abspath(os.path.join(
        _HERE,
        "..",
        "dualms2.0-benchmark",
        "network",
        "wan-performance-table8",
        "table8-summary.json",
    )),
]


def load_table8_json(path):
    if not os.path.isfile(path):
        raise ValueError("missing Table 8 summary: %s" % path)

    try:
        with open(path, "r", encoding="utf-8") as handle:
            data = json.load(handle)
    except (IOError, ValueError) as error:
        raise ValueError("cannot read %s: %s" % (path, error))

    schemes = data.get("schemes") if isinstance(data, dict) else None
    if not isinstance(schemes, list) or len(schemes) != 1:
        raise ValueError("%s: expected one Table 8 scheme" % path)

    scheme = schemes[0]
    try:
        sign1 = scheme["rounds"]["sign1"]
        sign2 = scheme["rounds"]["sign2"]
        return {
            "scheme": str(scheme["scheme"]),
            "runs": int(scheme["runs"]),
            "sign1_local": float(sign1["local_compute_ms"]),
            "sign1_network": float(sign1["network_latency_ms"]),
            "sign1_e2e": float(sign1["end_to_end_ms"]),
            "sign2_local": float(sign2["local_compute_ms"]),
            "sign2_network": float(sign2["network_latency_ms"]),
            "sign2_combine": float(sign2["combine_ms"]),
            "sign2_e2e": float(sign2["end_to_end_ms"]),
            "t": float(scheme["mean_rejected_attempts"]),
            "traffic_mib": float(scheme["mean_traffic_mib"]),
        }
    except (KeyError, TypeError, ValueError) as error:
        raise ValueError("%s: invalid Table 8 summary structure" % path)


def fmt(value, digits=3):
    return ("%%.%df" % digits) % value


def render_table(summaries):
    if len(summaries) != 2:
        raise ValueError("expected DualMS and rounded DualMS2.0 summaries")

    headers = [
        "Scheme",
        "Round",
        "Local",
        "Network",
        "Combine",
        "E2E",
        "t",
        "Traffic",
    ]
    row_groups = []
    for summary in summaries:
        row_groups.append([
            [
                summary["scheme"],
                "Sign1",
                fmt(summary["sign1_local"]),
                fmt(summary["sign1_network"]),
                "-",
                fmt(summary["sign1_e2e"]),
                fmt(summary["t"], 1),
                fmt(summary["traffic_mib"]),
            ],
            [
                "",
                "Sign2",
                fmt(summary["sign2_local"]),
                fmt(summary["sign2_network"]),
                fmt(summary["sign2_combine"]),
                fmt(summary["sign2_e2e"]),
                "",
                "",
            ],
        ])

    widths = [len(header) for header in headers]
    for group in row_groups:
        for row in group:
            for index, value in enumerate(row):
                widths[index] = max(widths[index], len(value))

    def border():
        return "+" + "+".join("-" * (width + 2) for width in widths) + "+"

    def print_row(row):
        print("|" + "|".join(
            " %-*s " % (widths[index], value)
            for index, value in enumerate(row)
        ) + "|")

    print("Table 8 WAN (ms; traffic in MiB)")
    for summary in summaries:
        print("%s: %d runs" % (summary["scheme"], summary["runs"]))
    print(border())
    print_row(headers)
    print(border())
    for group_index, group in enumerate(row_groups):
        for row in group:
            print_row(row)
        if group_index + 1 < len(row_groups):
            print(border())
    print(border())
    print("t: mean rejected attempts; Traffic: mean coordinator traffic.")


summary_paths = sys.argv[1:] if len(sys.argv) > 1 else DEFAULT_SUMMARIES
if len(summary_paths) != 2:
    sys.stderr.write("usage: sage table8.sage [DUALMS_JSON ROUNDING_JSON]\n")
    sys.exit(2)

try:
    summaries = [load_table8_json(path) for path in summary_paths]
    render_table(summaries)
except ValueError as error:
    sys.stderr.write("table8.sage: %s\n" % error)
    sys.exit(1)
