import contextlib
import io
import os

try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)

# Reuse the component calculations used by Table 6.  The two external
# schemes below are fixed values from the paper's Table 1.
print("Computing Table 1; Sage security estimates may take a few minutes...")
with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
    TABLE6_MODE = True

    load("dualms.sage")
    DUALMS_WITHOUT_ROUNDING_RESULTS = RESULTS[:]

    DUALMS_ROUNDING_NO_AUTORUN = True
    load("dualms-rounding.sage")
    DUALMS_WITH_ROUNDING_RESULTS = rounding_results()

    load("dualms2.0-non-rounding.sage")
    DUALMS2_NONROUNDING_RESULTS = RESULTS[:]

    load("dualms2.0-rounding.sage")
    DUALMS2_ROUNDING_RESULTS = RESULTS[:]


def fmt2(value):
    return ("%.2f" % RR(value).n())


def result_pair(results, key):
    return [fmt2(results[0][key]), fmt2(results[1][key])]


def signer_pair(results):
    return [str(results[0]["n"]), str(results[1]["n"])]


MULSIG_L = {
    "aggpk_kb": ["26.00", "32.50"],
    "aggsig_kb": ["98.47", "106.53"],
    "pk_kb": ["26.00", "32.50"],
    "sig_i_kb": ["-", "-"],
    "msg1_kb": ["-", "-"],
    "n": ["32", "1024"],
}

DILITHIUM_G = {
    "aggpk_kb": ["-"],
    "aggsig_kb": ["-"],
    "pk_kb": ["1.30"],
    "sig_i_kb": ["2.40"],
    "msg1_kb": ["-"],
    "n": ["-"],
}


GROUPS = [
    ("MulSig-L", "", ["n=32", "n=1024"]),
    ("DualMS", "without rounding", ["n=32", "n=1024"]),
    ("DualMS", "with rounding", ["n=32", "n=1024"]),
    ("DualMS2.0", "without rounding", ["n=32", "n=1024"]),
    ("DualMS2.0", "with rounding", ["n=32", "n=1024"]),
    ("Dilithium-G", "", [""]),
]


def table1_rows():
    computed = [
        DUALMS_WITHOUT_ROUNDING_RESULTS,
        DUALMS_WITH_ROUNDING_RESULTS,
        DUALMS2_NONROUNDING_RESULTS,
        DUALMS2_ROUNDING_RESULTS,
    ]
    rows = []
    for label, key in [
        ("|aggpk|", "aggpk_kb"),
        ("|AggSig|", "aggsig_kb"),
        ("|pk|", "pk_kb"),
        ("|Sig|", "sig_i_kb"),
        ("|msg|", "msg1_kb"),
    ]:
        rows.append([
            label,
            MULSIG_L[key],
            result_pair(computed[0], key),
            result_pair(computed[1], key),
            result_pair(computed[2], key),
            result_pair(computed[3], key),
            DILITHIUM_G[key],
        ])
    rows.append([
        "# of signers",
        MULSIG_L["n"],
        signer_pair(DUALMS_WITHOUT_ROUNDING_RESULTS),
        signer_pair(DUALMS_WITH_ROUNDING_RESULTS),
        signer_pair(DUALMS2_NONROUNDING_RESULTS),
        signer_pair(DUALMS2_ROUNDING_RESULTS),
        DILITHIUM_G["n"],
    ])
    return rows


def print_table1(groups, rows):
    metric_width = max(len("Scheme"), *(len(str(row[0])) for row in rows))
    group_subwidths = []
    group_widths = []

    for group_index, (_title, qualifier, subheaders) in enumerate(groups):
        subwidths = []
        for sub_index, subheader in enumerate(subheaders):
            values = [subheader]
            values.extend(str(row[group_index + 1][sub_index]) for row in rows)
            subwidths.append(max(len(value) for value in values))
        group_subwidths.append(subwidths)
        minimum_width = sum(subwidths) + 2 + 2 * (len(subwidths) - 1)
        group_widths.append(max(
            len(groups[group_index][0]) + 2,
            len(qualifier) + 2,
            minimum_width,
        ))

    def border():
        return "+" + "-" * (metric_width + 2) + "+" + "+".join(
            "-" * width for width in group_widths
        ) + "+"

    def group_cell(values, subwidths, group_width):
        fields = [str(value).rjust(subwidths[index]) for index, value in enumerate(values)]
        content = "  ".join(fields)
        inner_width = group_width - 2
        return " " + content.ljust(inner_width) + " "

    def centered_group_cell(value, group_width):
        return " " + str(value).center(group_width - 2) + " "

    def title_row():
        cells = [
            centered_group_cell(title, group_widths[index])
            for index, (title, _qualifier, _headers) in enumerate(groups)
        ]
        return "| " + "Scheme".ljust(metric_width) + " |" + "|".join(cells) + "|"

    def qualifier_row():
        cells = [
            centered_group_cell(qualifier, group_widths[index])
            for index, (_title, qualifier, _headers) in enumerate(groups)
        ]
        return "| " + "".ljust(metric_width) + " |" + "|".join(cells) + "|"

    def subheader_row():
        cells = [
            group_cell(headers, group_subwidths[index], group_widths[index])
            for index, (_title, _qualifier, headers) in enumerate(groups)
        ]
        return "| " + "".ljust(metric_width) + " |" + "|".join(cells) + "|"

    def data_row(row):
        cells = [
            group_cell(row[index + 1], group_subwidths[index], group_widths[index])
            for index in range(len(groups))
        ]
        return "| " + str(row[0]).ljust(metric_width) + " |" + "|".join(cells) + "|"

    print("Table 1 size comparison (KB)")
    print(border())
    print(title_row())
    print(qualifier_row())
    print(subheader_row())
    print(border())
    for row in rows:
        print(data_row(row))
    print(border())


print_table1(GROUPS, table1_rows())
