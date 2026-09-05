from sage.all import *
import contextlib
import io
import os

try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)
load("dualms_common.sage")

KB = 8 * 1024
N_STAR = 1

PROFILES = [
    {
        "name": "dualms-n32",
        "column_original": "n=32 original",
        "column_rounding": "n=32 rounding",
        "n": 32,
        "N": 256,
        "logq": 37,
        "k": 6,
        "ell": 7,
        "eta": 1,
        "eta_prime": 1,
        "alpha_coeff": 8.5,
        "kappa": 60,
        "gamma": 1.1,
        "challenge_bytes": 32,
        "nu_b": 11,
        "nu_w": 26,
    },
    {
        "name": "dualms-n1024",
        "column_original": "n=1024 original",
        "column_rounding": "n=1024 rounding",
        "n": 1024,
        "N": 256,
        "logq": 48,
        "k": 7,
        "ell": 9,
        "eta": 1,
        "eta_prime": 1,
        "alpha_coeff": 8.5,
        "kappa": 60,
        "gamma": 1.1,
        "challenge_bytes": 32,
        "nu_b": 11,
        "nu_w": 28,
    },
]

MLWE_BITS_CACHE = {}
MSIS_BITS_CACHE = {}


def cached_mlwe_classical_bits(eta, N, ell, logq, q):
    key = (Integer(eta), Integer(N), Integer(ell), Integer(logq), Integer(q))
    if key not in MLWE_BITS_CACHE:
        with contextlib.redirect_stdout(io.StringIO()):
            with contextlib.redirect_stderr(io.StringIO()):
                MLWE_BITS_CACHE[key] = get_classical_hardness(
                    findMLWEdelta(eta, N, ell, logq, q)
                )
    return MLWE_BITS_CACHE[key]


def cached_msis_classical_bits(gamma_bound, N, k, logq):
    key = (
        Integer(N),
        Integer(k),
        Integer(logq),
        "%.80f" % RR(log(gamma_bound, 2)),
    )
    if key not in MSIS_BITS_CACHE:
        MSIS_BITS_CACHE[key] = get_classical_hardness(
            findMSISdelta(gamma_bound, N, k, logq)
        )
    return MSIS_BITS_CACHE[key]


def fmt(x, digits=2):
    return ("%%.%df" % digits) % RR(x).n()


def fmt_delta(delta, digits=2):
    sign = "+" if RR(delta) >= 0 else ""
    return sign + fmt(delta, digits)


def fmt_delta_cell(value, baseline, digits=2):
    ratio = RR(value) / RR(baseline)
    return "%s (%s, %sx)" % (
        fmt(value, digits),
        fmt_delta(RR(value) - RR(baseline), digits),
        fmt(ratio, 2),
    )


def fmt_int_delta_cell(value, baseline):
    delta = Integer(value) - Integer(baseline)
    sign = "+" if delta >= 0 else ""
    ratio = RR(value) / RR(baseline)
    return "%s (%s%s, %sx)" % (
        value,
        sign,
        delta,
        fmt(ratio, 2),
    )


def fmt_float_with_delta(value, baseline, digits=1):
    return "%s (%s)" % (
        fmt(value, digits),
        fmt_delta(RR(value) - RR(baseline), digits),
    )


def fmt_power_with_delta(value, baseline, digits=3):
    return "%s (%s)" % (
        fmt(value, digits),
        fmt_delta(RR(value) - RR(baseline), digits),
    )


def print_table(title, headers, rows):
    widths = [
        max(len(str(header)), *[len(str(row[i])) for row in rows])
        for i, header in enumerate(headers)
    ]

    def line(left, fill, sep, right):
        return left + sep.join(fill * (width + 2) for width in widths) + right

    def row(values):
        cells = [
            " " + str(value).ljust(widths[i]) + " "
            for i, value in enumerate(values)
        ]
        return "|" + "|".join(cells) + "|"

    print(title)
    print(line("+", "-", "+", "+"))
    print(row(headers))
    print(line("+", "-", "+", "+"))
    for values in rows:
        print(row(values))
    print(line("+", "-", "+", "+"))
    print("")


def dualms_source_review_lines():
    return [
        "Paper review source: 2023-263 (4).pdf, Section 5, pp.21-22.",
        "DualMS paper assumptions used here: N=256, kappa=60, gamma=1.1, alpha=8.5*n, ell' = ell + 1, eta' = eta, challenge = 32 bytes.",
        "Table 2 parameter choices: n=32 uses q_bits=37, k=6, ell=7; n=1024 uses q_bits=48, k=7, ell=9.",
        "Implementation review: the baseline formulas are inherited from dualms.ipynb / dualms.sage, but are cleaned up here for consistent comparison.",
        "Terminology review: DualMS Table 2 'Sig' is the aggregated signature size; in CCS main Table 6 this corresponds to 'AggSig'.",
        "The extra 'sig' column in CCS main Table 6 is the per-signer second-round message size, so it is a different counting scope.",
        "Review fix applied here: the old n=32 legacy 8*kappa multiplier is not used; gamma/MSIS are computed directly from the norm bound.",
    ]


def print_dualms_source_review():
    print("DualMS source review")
    for line in dualms_source_review_lines():
        print("- " + line)
    print("")


def dualms_core_quantities(profile, use_rounding_q=False):
    n = Integer(profile["n"])
    N = Integer(profile["N"])
    logq = Integer(profile["logq"])
    k = Integer(profile["k"])
    ell = Integer(profile["ell"])
    ell_prime = ell + 1
    eta = Integer(profile["eta"])
    eta_prime = Integer(profile["eta_prime"])
    alpha = RR(profile["alpha_coeff"]) * n
    kappa = Integer(profile["kappa"])
    gamma = RR(profile["gamma"])
    challenge_bytes = Integer(profile["challenge_bytes"])
    nu_b = Integer(profile["nu_b"])
    nu_w = Integer(profile["nu_w"])

    rounding_nus = [nu_b, nu_w] if use_rounding_q else None
    q = find_prime_q_below_power(
        logq, modulus=8, residue=5, rounding_nus=rounding_nus
    )
    q_bits = q_bit_length(q)
    m = ell + k
    m_prime = ell_prime + k

    T = kappa * kappa * eta * sqrt(N * m)
    T_prime = kappa * kappa * eta_prime * sqrt(N * m_prime)
    sigma = max(
        sqrt(2 * pi) * alpha * T,
        2 * N * (q ^ (k / m + 2 / (N * m))),
    )
    sigma_prime = max(
        sqrt(2 * pi) * alpha * T_prime,
        2 * N * (q ^ (k / m_prime + 2 / (N * m_prime))),
    )

    B_z = gamma * (sigma / sqrt(2 * pi)) * sqrt(N * ell)
    B_n = sqrt(n) * B_z
    B_r = gamma * (sigma_prime / sqrt(2 * pi)) * sqrt(N * ell_prime)
    B_n_prime = sqrt(n) * B_r
    B_e = gamma * (sqrt(sigma ^ 2 + sigma_prime ^ 2) / sqrt(2 * pi)) * sqrt(N * k)
    B_n_double_prime = sqrt(n) * B_e

    pk_kb = N * k * q_bits / KB
    msg1_kb = pk_kb
    c_bits = challenge_bytes * 8

    B_z1 = gamma * (sigma / sqrt(2 * pi)) * sqrt(N * m)
    B_r1 = gamma * (sigma_prime / sqrt(2 * pi)) * sqrt(N * m_prime)
    sig_i_kb = (
        c_bits
        + m * N * (1 + ceil(log(B_z1 / sqrt(m * N), 2)))
        + m_prime * N * (1 + ceil(log(B_r1 / sqrt(m_prime * N), 2)))
    ) / KB

    b_n = B_n / sqrt(ell * N)
    b_n_prime = B_n_prime / sqrt(ell_prime * N)
    b_n_double_prime = B_n_double_prime / sqrt(k * N)
    aggsig_kb = (
        c_bits
        + ell * N * (1 + ceil(log(b_n, 2)))
        + ell_prime * N * (1 + ceil(log(b_n_prime, 2)))
        + k * N * (1 + ceil(log(b_n_double_prime, 2)))
    ) / KB

    return {
        "profile": profile["name"],
        "column_original": profile["column_original"],
        "column_rounding": profile["column_rounding"],
        "n": n,
        "N": N,
        "logq": logq,
        "q": q,
        "q_bits": q_bits,
        "k": k,
        "ell": ell,
        "ell_prime": ell_prime,
        "eta": eta,
        "eta_prime": eta_prime,
        "alpha": alpha,
        "kappa": kappa,
        "gamma": gamma,
        "challenge_bytes": challenge_bytes,
        "nu_b": nu_b,
        "nu_w": nu_w,
        "m": m,
        "m_prime": m_prime,
        "sigma": sigma,
        "sigma_prime": sigma_prime,
        "B_n": B_n,
        "B_n_prime": B_n_prime,
        "B_n_double_prime": B_n_double_prime,
        "pk_kb": pk_kb,
        "msg1_kb": msg1_kb,
        "sig_i_kb": sig_i_kb,
        "aggsig_kb": aggsig_kb,
    }


def dualms_original_result(profile):
    values = dualms_core_quantities(profile, use_rounding_q=False)
    gamma_original = sqrt(
        values["B_n"] ^ 2
        + values["B_n_prime"] ^ 2
        + values["B_n_double_prime"] ^ 2
    )
    mlwe_c = cached_mlwe_classical_bits(
        values["eta"], values["N"], values["ell"], values["logq"], values["q"]
    )
    msis_c = cached_msis_classical_bits(
        gamma_original, values["N"], values["k"], values["logq"]
    )

    result = dict(values)
    result.update({
        "aggpk_kb": values["pk_kb"],
        "aggpk_plus_aggsig_kb": values["pk_kb"] + values["aggsig_kb"],
        "log_sigma": log(values["sigma_prime"], 2),
        "log_gamma": log(gamma_original, 2),
        "msis_c": msis_c,
        "mlwe_c": mlwe_c,
    })
    return result


def dualms_rounding_result(profile, original=None):
    values = dualms_core_quantities(profile, use_rounding_q=True)
    N = values["N"]
    k = values["k"]
    nu_b = values["nu_b"]
    nu_w = values["nu_w"]
    kappa = values["kappa"]
    q = values["q"]

    q_b = Integer(floor(q / (2 ^ nu_b)))
    q_b_bpc = 0 if q_b <= 1 else ceil(log(q_b, 2))
    q_h = max(1, Integer(floor(q / (2 ^ nu_w))))
    q_h_bpc = 0 if q_h <= 1 else ceil(log(q_h, 2))
    q_h_cap = max(0, floor((q_h - 1) / 2))

    B_h = values["B_n_double_prime"] + sqrt(k * N) * (
        kappa * 2 ^ (nu_b - 1) + 2 ^ nu_w
    )
    gamma_rounding = sqrt(
        values["B_n"] ^ 2
        + values["B_n_prime"] ^ 2
        + B_h ^ 2
        + (N_STAR ^ 2) * kappa
        + (2 ^ (2 * nu_b)) * N * k
    )

    h_amp = max(0, floor(B_h / (2 ^ nu_w)))
    h_bpc = q_h_bpc if h_amp >= q_h_cap else (
        0 if h_amp == 0 else ceil(log(2 * h_amp + 1, 2))
    )

    aggpk_kb = k * N * q_b_bpc / KB
    aggsig_kb = (
        values["challenge_bytes"] * 8
        + values["ell"] * N * (1 + ceil(log(values["B_n"] / sqrt(values["ell"] * N), 2)))
        + values["ell_prime"] * N * (1 + ceil(log(values["B_n_prime"] / sqrt(values["ell_prime"] * N), 2)))
        + k * N * h_bpc
    ) / KB

    mlwe_c = cached_mlwe_classical_bits(
        values["eta"], values["N"], values["ell"], values["logq"], values["q"]
    )
    msis_c = cached_msis_classical_bits(
        gamma_rounding, values["N"], values["k"], values["logq"]
    )

    result = dict(values)
    result.update({
        "aggpk_kb": aggpk_kb,
        "q_b_bpc": q_b_bpc,
        "q_h_bpc": q_h_bpc,
        "h_bpc": h_bpc,
        "aggsig_kb": aggsig_kb,
        "aggpk_plus_aggsig_kb": aggpk_kb + aggsig_kb,
        "saving_kb": None if original is None else (
            original["aggpk_plus_aggsig_kb"] - (aggpk_kb + aggsig_kb)
        ),
        "log_sigma": log(values["sigma_prime"], 2),
        "log_B_h": log(B_h, 2),
        "log_gamma": log(gamma_rounding, 2),
        "msis_c": msis_c,
        "mlwe_c": mlwe_c,
    })
    return result


def original_results():
    return [dualms_original_result(profile) for profile in PROFILES]


def rounding_results():
    baseline = original_results()
    return [
        dualms_rounding_result(profile, original=baseline[index])
        for index, profile in enumerate(PROFILES)
    ]


def dualms_original_rows(results):
    return [
        ["n", str(results[0]["n"]), str(results[1]["n"])],
        ["N", str(results[0]["N"]), str(results[1]["N"])],
        ["q_bits", str(results[0]["q_bits"]), str(results[1]["q_bits"])],
        ["k", str(results[0]["k"]), str(results[1]["k"])],
        ["ell", str(results[0]["ell"]), str(results[1]["ell"])],
        ["ell' = ell + 1", str(results[0]["ell_prime"]), str(results[1]["ell_prime"])],
        ["eta", str(results[0]["eta"]), str(results[1]["eta"])],
        ["kappa", str(results[0]["kappa"]), str(results[1]["kappa"])],
        ["gamma", fmt(results[0]["gamma"], 1), fmt(results[1]["gamma"], 1)],
        ["alpha coeff", "8.5 * n", "8.5 * n"],
        ["challenge bytes", str(results[0]["challenge_bytes"]), str(results[1]["challenge_bytes"])],
        ["pk KB", fmt(results[0]["pk_kb"]), fmt(results[1]["pk_kb"])],
        ["aggpk KB (CCS Table 6 view)", fmt(results[0]["aggpk_kb"]), fmt(results[1]["aggpk_kb"])],
        ["msg1 KB", fmt(results[0]["msg1_kb"]), fmt(results[1]["msg1_kb"])],
        ["sig_i KB (CCS Table 6 'sig')", fmt(results[0]["sig_i_kb"]), fmt(results[1]["sig_i_kb"])],
        ["aggsig KB (= DualMS Table 2 'Sig')", fmt(results[0]["aggsig_kb"]), fmt(results[1]["aggsig_kb"])],
        ["aggpk+aggsig KB", fmt(results[0]["aggpk_plus_aggsig_kb"]), fmt(results[1]["aggpk_plus_aggsig_kb"])],
        ["log2(s')", fmt(results[0]["log_sigma"], 3), fmt(results[1]["log_sigma"], 3)],
        ["MSIS bits", fmt(results[0]["msis_c"], 1), fmt(results[1]["msis_c"], 1)],
        ["MLWE bits", fmt(results[0]["mlwe_c"], 1), fmt(results[1]["mlwe_c"], 1)],
    ]


def print_dualms_original_ccs_table():
    results = original_results()
    print_table(
        "Original DualMS baseline recomputed from 2023-263 Section 5",
        ["metric", results[0]["column_original"], results[1]["column_original"]],
        dualms_original_rows(results),
    )


def dualms_rounding_comparison_rows(original, rounded):
    return [
        ["n", str(original[0]["n"]), str(rounded[0]["n"]), str(original[1]["n"]), str(rounded[1]["n"])],
        ["N", str(original[0]["N"]), str(rounded[0]["N"]), str(original[1]["N"]), str(rounded[1]["N"])],
        ["q_bits", str(original[0]["q_bits"]), str(rounded[0]["q_bits"]), str(original[1]["q_bits"]), str(rounded[1]["q_bits"])],
        ["k", str(original[0]["k"]), str(rounded[0]["k"]), str(original[1]["k"]), str(rounded[1]["k"])],
        ["ell", str(original[0]["ell"]), str(rounded[0]["ell"]), str(original[1]["ell"]), str(rounded[1]["ell"])],
        ["ell' = ell + 1", str(original[0]["ell_prime"]), str(rounded[0]["ell_prime"]), str(original[1]["ell_prime"]), str(rounded[1]["ell_prime"])],
        ["nu_b", "--", str(rounded[0]["nu_b"]), "--", str(rounded[1]["nu_b"])],
        ["nu_w", "--", str(rounded[0]["nu_w"]), "--", str(rounded[1]["nu_w"])],
        ["pk KB", fmt(original[0]["pk_kb"]), fmt_delta_cell(rounded[0]["pk_kb"], original[0]["pk_kb"]), fmt(original[1]["pk_kb"]), fmt_delta_cell(rounded[1]["pk_kb"], original[1]["pk_kb"])],
        ["aggpk KB", fmt(original[0]["aggpk_kb"]), fmt_delta_cell(rounded[0]["aggpk_kb"], original[0]["aggpk_kb"]), fmt(original[1]["aggpk_kb"]), fmt_delta_cell(rounded[1]["aggpk_kb"], original[1]["aggpk_kb"])],
        ["msg1 KB", fmt(original[0]["msg1_kb"]), fmt_delta_cell(rounded[0]["msg1_kb"], original[0]["msg1_kb"]), fmt(original[1]["msg1_kb"]), fmt_delta_cell(rounded[1]["msg1_kb"], original[1]["msg1_kb"])],
        ["sig_i KB (CCS Table 6 'sig')", fmt(original[0]["sig_i_kb"]), fmt_delta_cell(rounded[0]["sig_i_kb"], original[0]["sig_i_kb"]), fmt(original[1]["sig_i_kb"]), fmt_delta_cell(rounded[1]["sig_i_kb"], original[1]["sig_i_kb"])],
        ["aggsig KB (= DualMS Table 2 'Sig')", fmt(original[0]["aggsig_kb"]), fmt_delta_cell(rounded[0]["aggsig_kb"], original[0]["aggsig_kb"]), fmt(original[1]["aggsig_kb"]), fmt_delta_cell(rounded[1]["aggsig_kb"], original[1]["aggsig_kb"])],
        ["aggpk+aggsig KB", fmt(original[0]["aggpk_plus_aggsig_kb"]), fmt_delta_cell(rounded[0]["aggpk_plus_aggsig_kb"], original[0]["aggpk_plus_aggsig_kb"]), fmt(original[1]["aggpk_plus_aggsig_kb"]), fmt_delta_cell(rounded[1]["aggpk_plus_aggsig_kb"], original[1]["aggpk_plus_aggsig_kb"])],
        ["h bpc", "--", str(rounded[0]["h_bpc"]), "--", str(rounded[1]["h_bpc"])],
        ["saving KB", "0.00", fmt(rounded[0]["saving_kb"]), "0.00", fmt(rounded[1]["saving_kb"])],
        ["log2(s')", fmt(original[0]["log_sigma"], 3), fmt_power_with_delta(rounded[0]["log_sigma"], original[0]["log_sigma"], 3), fmt(original[1]["log_sigma"], 3), fmt_power_with_delta(rounded[1]["log_sigma"], original[1]["log_sigma"], 3)],
        ["log2(B_h)", "--", fmt(rounded[0]["log_B_h"], 4), "--", fmt(rounded[1]["log_B_h"], 4)],
        ["log2(gamma)", fmt(original[0]["log_gamma"], 4), fmt_power_with_delta(rounded[0]["log_gamma"], original[0]["log_gamma"], 4), fmt(original[1]["log_gamma"], 4), fmt_power_with_delta(rounded[1]["log_gamma"], original[1]["log_gamma"], 4)],
        ["MSIS bits", fmt(original[0]["msis_c"], 1), fmt_float_with_delta(rounded[0]["msis_c"], original[0]["msis_c"], 1), fmt(original[1]["msis_c"], 1), fmt_float_with_delta(rounded[1]["msis_c"], original[1]["msis_c"], 1)],
        ["MLWE bits", fmt(original[0]["mlwe_c"], 1), fmt_float_with_delta(rounded[0]["mlwe_c"], original[0]["mlwe_c"], 1), fmt(original[1]["mlwe_c"], 1), fmt_float_with_delta(rounded[1]["mlwe_c"], original[1]["mlwe_c"], 1)],
    ]


def print_dualms_rounding_comparison():
    original = original_results()
    rounded = rounding_results()
    print_table(
        "DualMS + rounding comparison",
        [
            "metric",
            PROFILES[0]["column_original"],
            PROFILES[0]["column_rounding"],
            PROFILES[1]["column_original"],
            PROFILES[1]["column_rounding"],
        ],
        dualms_rounding_comparison_rows(original, rounded),
    )


def run_all():
    print_dualms_source_review()
    print_dualms_original_ccs_table()
    print_dualms_rounding_comparison()


def _should_autorun():
    if globals().get("DUALMS_ROUNDING_NO_AUTORUN", False):
        return False
    source = globals().get("__file__", "")
    return source.endswith("dualms-rounding.sage.py")


if _should_autorun():
    run_all()
