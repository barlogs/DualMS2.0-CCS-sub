import os
try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)
load("dualms_common.sage")

KB = 8 * 1024
N_STAR = 1

DUALMS_PROFILE = {
    "name": "dualms-n8", "n": 8, "N": 256, "logq": 37,
    "k": 5, "ell": 7, "eta": 1, "eta_prime": 1,
    "alpha_coeff": 8.5, "kappa": 60, "gamma": 1.1,
    "challenge_bytes": 32,
}

ROUNDING_PROFILE = {
    "name": "dualms2.0-rounding-n8", "n": 8, "N": 256,
    "m": 6, "ell": 6, "eta": 1, "kappa": 60,
    "challenge_bytes": 32, "logq": 32, "nu_b": 11, "nu_w": 26,
    "alpha_coeff": 4.917,
}

def dualms_result(profile):
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
    q_bits = q_bit_length(q)
    m = ell + k
    m_prime = ell_prime + k
    T = kappa * kappa * eta * sqrt(N * m)
    T_prime = kappa * kappa * eta_prime * sqrt(N * m_prime)
    s = max(
        sqrt(2 * pi) * alpha * T,
        2 * N * q ^ (k / m + 2 / (N * m)),
    )
    s_prime = max(
        sqrt(2 * pi) * alpha * T_prime,
        2 * N * q ^ (k / m_prime + 2 / (N * m_prime)),
    )

    B = gamma * (s / sqrt(2 * pi)) * sqrt(N * ell)
    B_n = sqrt(n) * B
    B_prime = gamma * (s_prime / sqrt(2 * pi)) * sqrt(N * ell_prime)
    B_n_prime = sqrt(n) * B_prime
    B_prime_prime = gamma * (sqrt(s ^ 2 + s_prime ^ 2) / sqrt(2 * pi)) * sqrt(N * k)
    B_n_prime_prime = sqrt(n) * B_prime_prime

    size_c_bits = challenge_bytes * 8
    pk_kb = N * k * q_bits / KB
    msg1_kb = pk_kb

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

    mlwe_c = get_classical_hardness(findMLWEdelta(eta, N, ell, logq, q))
    msis_norm = sqrt(B_n ^ 2 + B_n_prime ^ 2 + B_n_prime_prime ^ 2)
    msis_c = get_classical_hardness(findMSISdelta(msis_norm, N, k, logq))

    return {
        "profile": profile["name"], "n": n, "q_bits": q_bits,
        "pk_kb": pk_kb, "aggpk_kb": pk_kb, "msg1_kb": msg1_kb,
        "sig_i_kb": sig_i_kb, "aggsig_kb": aggsig_kb,
        "pk_plus_sig_kb": pk_kb + aggsig_kb, "log_sigma": log(s_prime, 2),
        "msis_c": msis_c, "mlwe_c": mlwe_c,
    }

def rounding_result(profile):
    n = profile["n"]
    N = profile["N"]
    m = profile["m"]
    ell = profile["ell"]
    eta = profile["eta"]
    kappa = profile["kappa"]
    challenge_bytes = profile["challenge_bytes"]
    logq = profile["logq"]
    nu_b = profile["nu_b"]
    nu_w = profile["nu_w"]
    q = find_prime_q_below_power(logq, modulus=8, residue=5, rounding_nus=[nu_b, nu_w])
    q_bits = q_bit_length(q)

    alpha = profile["alpha_coeff"] * n
    T = kappa ^ 2 * eta * sqrt(N * (ell + m + 1))
    sigma = max(
        alpha * T,
        2 * N * q ^ (m / (m + ell) + 2 / (N * (m + ell))) / sqrt(2 * pi),
    )

    dt = 2 ^ (nu_b - 1)
    Hz = sigma * sqrt(2 * n * ell * N)
    Hh = sqrt(m * N) * (sigma * sqrt(2 * n) + eta + kappa * dt + 2 ^ nu_w)
    hR = Hh / (2 ^ nu_w)
    B = Hz + Hh
    B_n_prime = sigma * sqrt(2 * n * N)
    B_n_double_prime = eta * sqrt(ell * N * n)
    gamma2 = sqrt(
        B ^ 2 + B_n_prime ^ 2 + B_n_double_prime ^ 2 +
        (N_STAR ^ 2) * kappa + (2 ^ (2 * nu_b)) * N * m
    )

    c_bits = challenge_bytes * 8
    ent = RR(1) / 2 + log(sqrt(2 * pi), 2)
    qb = Integer(floor(q / (2 ^ nu_b)))
    qb_bpc = 0 if qb <= 1 else ceil(log(qb, 2))
    qh = max(1, Integer(floor(q / (2 ^ nu_w))))
    qh_bpc = 0 if qh <= 1 else ceil(log(qh, 2))
    qh_cap = max(0, floor((qh - 1) / 2))

    pk_kb = m * N * q_bits / KB
    aggpk_kb = m * N * qb_bpc / KB
    msg1_kb = pk_kb
    sig_i_kb = (
        c_bits +
        (ell + m) * N * (ent + log(sigma, 2)) +
        N * (ent + log(sigma, 2)) +
        (ell + m) * N * ceil(log(3, 2))
    ) / KB

    h_amp = max(0, floor(hR))
    h_bpc = qh_bpc if h_amp >= qh_cap else (0 if h_amp == 0 else ceil(log(2 * h_amp + 1, 2)))
    aggsig_kb = (
        c_bits +
        ell * N * (ent + log(sqrt(n) * sigma, 2)) +
        N * (ent + log(sqrt(n) * sigma, 2)) +
        ell * N * ceil(log(2 * n * eta + 1, 2)) +
        m * N * h_bpc
    ) / KB

    mlwe_c = get_classical_hardness(findMLWEdelta(eta, N, ell, logq, q))
    msis_c = get_classical_hardness(findMSISdelta(gamma2, N, m, logq))

    return {
        "profile": profile["name"], "n": n, "q_bits": q_bits,
        "pk_kb": pk_kb, "aggpk_kb": aggpk_kb, "msg1_kb": msg1_kb,
        "sig_i_kb": sig_i_kb, "aggsig_kb": aggsig_kb,
        "pk_plus_sig_kb": aggpk_kb + aggsig_kb, "log_sigma": log(sigma, 2),
        "msis_c": msis_c, "mlwe_c": mlwe_c,
    }

def fmt(x, digits=2):
    return ("%%.%df" % digits) % RR(x).n()

print("WAN benchmark n=8 parameter summary")
print("profile | n | q_bits | pk | aggpk | msg1 | sig | aggsig | aggpk+aggsig | log_sigma | MSIS | MLWE")
for result in [dualms_result(DUALMS_PROFILE), rounding_result(ROUNDING_PROFILE)]:
    print(
        result["profile"], "|", result["n"], "|", result["q_bits"], "|",
        fmt(result["pk_kb"]), "|", fmt(result["aggpk_kb"]), "|", fmt(result["msg1_kb"]), "|",
        fmt(result["sig_i_kb"]), "|", fmt(result["aggsig_kb"]), "|",
        fmt(result["pk_plus_sig_kb"]), "|", fmt(result["log_sigma"]), "|",
        fmt(result["msis_c"], 1), "|", fmt(result["mlwe_c"], 1)
    )
