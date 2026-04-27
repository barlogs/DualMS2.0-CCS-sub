import os
try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)
load("dualms_common.sage")

KB = 8 * 1024

PROFILES = [
    {
        "name": "dualms_n32", "n": 32, "N": 256, "logq": 37,
        "k": 6, "ell": 7, "eta": 1, "eta_prime": 1,
        "alpha_coeff": 8.5, "kappa": 60, "gamma": 1.1,
        "challenge_bytes": 32, "msis_multiplier_mode": "8kappa",
    },
    {
        "name": "dualms_n1024", "n": 1024, "N": 256, "logq": 48,
        "k": 7, "ell": 9, "eta": 1, "eta_prime": 1,
        "alpha_coeff": 8.5, "kappa": 60, "gamma": 1.1,
        "challenge_bytes": 32, "msis_multiplier_mode": "1",
    },
]

def profile_result(profile):
    n = profile["n"]
    N = profile["N"]
    logq = profile["logq"]
    k = profile["k"]
    ell = profile["ell"]
    ell_prime = ell + 1
    eta = profile["eta"]
    eta_prime = profile["eta_prime"]
    alpha = profile["alpha_coeff"] * n
    kappa = profile["kappa"]
    gamma = profile["gamma"]
    challenge_bytes = profile["challenge_bytes"]

    q = find_prime_q_below_power(logq, modulus=8, residue=5)
    q_bits = ceil(log(q, 2))
    m = ell + k
    m_prime = ell_prime + k
    T = kappa * kappa * eta * sqrt(N * m)
    T_prime = kappa * kappa * eta_prime * sqrt(N * m_prime)
    s = max(sqrt(2 * pi) * alpha * T, 2 * N * (q ^ (k / m + 2 / (N * m))))
    s_prime = max(sqrt(2 * pi) * alpha * T_prime, 2 * N * (q ^ (k / m_prime + 2 / (N * m_prime))))

    B = gamma * (s / sqrt(2 * pi)) * sqrt(N * ell)
    B_n = sqrt(n) * B
    B_prime = gamma * (s_prime / sqrt(2 * pi)) * sqrt(N * ell_prime)
    B_n_prime = sqrt(n) * B_prime
    B_prime_prime = gamma * (sqrt(s ^ 2 + s_prime ^ 2) / sqrt(2 * pi)) * sqrt(N * k)
    B_n_prime_prime = sqrt(n) * B_prime_prime

    size_c_bits = challenge_bytes * 8
    pk_kb = N * k * q_bits / KB
    msg1_kb = k * N * q_bits / KB

    b_n = B_n / sqrt(ell * N)
    b_n_prime = B_n_prime / sqrt(ell_prime * N)
    b_n_prime_prime = B_n_prime_prime / sqrt(k * N)
    aggsig_bits = (
        ell * N * (1 + ceil(log(b_n, 2))) +
        ell_prime * N * (1 + ceil(log(b_n_prime, 2))) +
        k * N * (1 + ceil(log(b_n_prime_prime, 2)))
    )
    aggsig_kb = (size_c_bits + aggsig_bits) / KB

    B_z1 = gamma * (s / sqrt(2 * pi)) * sqrt(N * m)
    B_r1 = gamma * (s_prime / sqrt(2 * pi)) * sqrt(N * m_prime)
    sig_i_bits = (
        size_c_bits +
        m * N * (1 + ceil(log(B_z1 / sqrt(m * N), 2))) +
        m_prime * N * (1 + ceil(log(B_r1 / sqrt(m_prime * N), 2)))
    )
    sig_i_kb = sig_i_bits / KB

    mlwe_delta = findMLWEdelta(eta, N, ell, logq, q)
    msis_norm = sqrt(B_n ^ 2 + B_n_prime ^ 2 + B_n_prime_prime ^ 2)
    msis_bound = msis_norm * (8 * kappa) if profile["msis_multiplier_mode"] == "8kappa" else msis_norm
    msis_delta = findMSISdelta(msis_bound, N, k, logq)

    return {
        "profile": profile["name"], "n": n, "q_bits": q_bits,
        "pk_kb": pk_kb, "msg1_kb": msg1_kb, "sig_i_kb": sig_i_kb,
        "aggsig_kb": aggsig_kb, "pk_plus_sig_kb": pk_kb + aggsig_kb,
        "log_sigma": log(s_prime, 2),
        "mlwe_c": get_classical_hardness(mlwe_delta),
        "msis_c": get_classical_hardness(msis_delta),
    }

def fmt(x, digits=2):
    return ("%%.%df" % digits) % RR(x).n()

print("DualMS parameter summary")
print("profile | n | q_bits | pk | msg1 | sig | aggsig | pk+aggsig | log_sigma | MSIS | MLWE")
for result in [profile_result(profile) for profile in PROFILES]:
    print(
        result["profile"], "|", result["n"], "|", result["q_bits"], "|",
        fmt(result["pk_kb"]), "|", fmt(result["msg1_kb"]), "|",
        fmt(result["sig_i_kb"]), "|", fmt(result["aggsig_kb"]), "|",
        fmt(result["pk_plus_sig_kb"]), "|", fmt(result["log_sigma"]), "|",
        fmt(result["msis_c"], 1), "|", fmt(result["mlwe_c"], 1)
    )
