import os
try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)
load("dualms_common.sage")

KB = 8 * 1024
DEFAULT_ALPHA_COEFF = 4.917

PROFILES = [
    {
        "name": "dualms2.0-non-rounding-n32", "n": 32, "N": 256,
        "m": 7, "ell": 6, "eta": 1, "kappa": 60,
        "challenge_bytes": 32, "logq": 31, "alpha_coeff": DEFAULT_ALPHA_COEFF,
    },
    {
        "name": "dualms2.0-non-rounding-n1024", "n": 1024, "N": 256,
        "m": 10, "ell": 6, "eta": 1, "kappa": 60,
        "challenge_bytes": 32, "logq": 32, "alpha_coeff": DEFAULT_ALPHA_COEFF,
    },
]

def profile_result(profile):
    n = profile["n"]
    N = profile["N"]
    m = profile["m"]
    ell = profile["ell"]
    eta = profile["eta"]
    kappa = profile["kappa"]
    challenge_bytes = profile["challenge_bytes"]
    logq = profile["logq"]
    q = find_prime_q_below_power(logq, modulus=8, residue=5)
    q_bits = q_bit_length(q)

    alpha = profile["alpha_coeff"] * n
    T = kappa ^ 2 * eta * sqrt(N * (ell + m + 1))
    sigma = max(
        alpha * T,
        2 * N * q ^ (m / (m + ell) + 2 / (N * (m + ell))) / sqrt(2 * pi),
    )
    sigma_eta = sqrt(((2 * eta + 1) ^ 2 - 1) / 12)
    ent = RR(1) / 2 + log(sqrt(2 * pi), 2)
    c_bits = challenge_bytes * 8

    pk_kb = m * N * q_bits / KB
    msg1_kb = pk_kb
    sig_i_kb = (
        c_bits +
        (ell + m) * N * (ent + log(sigma, 2)) +
        N * (ent + log(sigma, 2)) +
        (ell + m) * N * (ent + log(sigma_eta, 2))
    ) / KB

    sigma_agg = sqrt(sigma_eta ^ 2 + sigma ^ 2)
    aggsig_kb = (
        c_bits +
        ell * N * (ent + log(sqrt(n) * sigma, 2)) +
        N * (ent + log(sqrt(n) * sigma, 2)) +
        ell * N * (ent + log(sqrt(n) * sigma_eta, 2)) +
        m * N * (ent + log(sqrt(n) * sigma_agg, 2))
    ) / KB

    B = sqrt(2) * sigma * sqrt((m + ell) * N)
    B_n = sqrt(n) * B
    B_prime = sqrt(2) * sigma * sqrt(N)
    B_n_prime = sqrt(n) * B_prime
    B_double_prime = eta * sqrt((ell + m) * N)
    B_n_double_prime = sqrt(n) * B_double_prime
    gamma_bound = sqrt(2 * B_n ^ 2 + 2 * B_n_prime ^ 2 + 2 * B_n_double_prime ^ 2 + kappa ^ 3)

    mlwe_delta = findMLWEdelta(eta, N, ell, logq, q)
    msis_delta = findMSISdelta(gamma_bound, N, m, logq)

    return {
        "profile": profile["name"], "n": n, "q_bits": q_bits,
        "pk_kb": pk_kb, "msg1_kb": msg1_kb, "sig_i_kb": sig_i_kb,
        "aggsig_kb": aggsig_kb,
        "pk_plus_sig_kb": pk_kb + aggsig_kb, "log_sigma": log(sigma, 2),
        "mlwe_c": get_classical_hardness(mlwe_delta),
        "msis_c": get_classical_hardness(msis_delta),
    }

def fmt(x, digits=2):
    return ("%%.%df" % digits) % RR(x).n()

print("DualMS2.0 non-rounding parameter summary")
print("profile | n | q_bits | pk | msg1 | sig | aggsig | pk+aggsig | log_sigma | MSIS | MLWE")
for result in [profile_result(profile) for profile in PROFILES]:
    print(
        result["profile"], "|", result["n"], "|", result["q_bits"], "|",
        fmt(result["pk_kb"]), "|", fmt(result["msg1_kb"]), "|",
        fmt(result["sig_i_kb"]), "|", fmt(result["aggsig_kb"]), "|",
        fmt(result["pk_plus_sig_kb"]), "|", fmt(result["log_sigma"]), "|",
        fmt(result["msis_c"], 1), "|", fmt(result["mlwe_c"], 1)
    )
