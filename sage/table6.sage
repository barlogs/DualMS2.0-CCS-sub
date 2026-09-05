import os

try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)

TABLE6_MODE = True

load("dualms.sage")
DUALMS_RESULTS = RESULTS[:]
load("dualms2.0-non-rounding.sage")
NONROUNDING_RESULTS = RESULTS[:]
load("dualms2.0-rounding.sage")
ROUNDING_RESULTS = RESULTS[:]

ALL_RESULTS = DUALMS_RESULTS + NONROUNDING_RESULTS + ROUNDING_RESULTS

def fmt(x, digits=2):
    return ("%%.%df" % digits) % RR(x).n()

def values(key):
    return [result[key] for result in ALL_RESULTS]

def formatted_values(key, digits=2):
    return [fmt(result[key], digits) for result in ALL_RESULTS]

def optional_values(key):
    return [result.get(key, "--") for result in ALL_RESULTS]

headers = [
    "Parameter",
    "n=32", "n=1024",
    "n=32", "n=1024",
    "n=32", "n=1024",
]

scheme_headers = [
    "dualms",
    "dualms2.0-non-rounding",
    "dualms2.0-rounding",
]

parameter_rows = [
    ["n"] + values("n"),
    ["N"] + values("N"),
    ["ceil(log(q))"] + values("q_bits"),
    ["m"] + values("m"),
    ["ell"] + values("ell"),
    ["eta"] + values("eta"),
    ["nu_b"] + optional_values("nu_b"),
    ["nu_w"] + optional_values("nu_w"),
    ["|pk|"] + formatted_values("pk_kb"),
    ["|aggpk|"] + formatted_values("aggpk_kb"),
    ["|sig|"] + formatted_values("sig_i_kb"),
    ["|AggSig|"] + formatted_values("aggsig_kb"),
    ["|aggpk| + |AggSig|"] + formatted_values("pk_plus_sig_kb"),
    ["log(sigma)"] + formatted_values("log_sigma"),
    ["MSIS"] + formatted_values("msis_c", 1),
    ["MLWE"] + formatted_values("mlwe_c", 1),
]

def print_compact_table(headers, scheme_headers, rows):
    text_rows = [[str(value) for value in row] for row in rows]
    widths = [
        max(len(headers[index]), *(len(row[index]) for row in text_rows))
        for index in range(len(headers))
    ]
    groups = [(1, 2), (3, 4), (5, 6)]
    group_widths = [
        max(len(title) + 2, widths[left] + widths[right] + 4)
        for title, (left, right) in zip(scheme_headers, groups)
    ]

    def border():
        return "+" + "-" * (widths[0] + 2) + "+" + "+".join("-" * width for width in group_widths) + "+"

    def format_group(row, left, right, group_width):
        gap = group_width - widths[left] - widths[right] - 2
        return " " + row[left].rjust(widths[left]) + " " * gap + row[right].rjust(widths[right]) + " "

    def format_row(row):
        groups_text = [
            format_group(row, left, right, group_width)
            for (left, right), group_width in zip(groups, group_widths)
        ]
        return "| " + row[0].ljust(widths[0]) + " |" + "|".join(groups_text) + "|"

    def format_scheme_header():
        groups_text = [
            title.center(group_width)
            for title, group_width in zip(scheme_headers, group_widths)
        ]
        return "| " + headers[0].ljust(widths[0]) + " |" + "|".join(groups_text) + "|"

    print(border())
    print(format_scheme_header())
    print(format_row([""] + headers[1:]))
    print(border())
    for row in text_rows:
        print(format_row(row))
    print(border())

print("Table 6 parameter summary")
print_compact_table(headers, scheme_headers, parameter_rows)
