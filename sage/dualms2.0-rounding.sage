import os
try:
    _HERE = os.path.dirname(os.path.abspath(__file__))
except NameError:
    _HERE = os.getcwd()
os.chdir(_HERE)
load("dualms_common.sage")

KB = 8 * 1024
DEFAULT_ALPHA_COEFF = 4.917
N_STAR = 1

PROFILES = [
    {
        "name": "dualms2.0-rounding-n32", "n": 32, "N": 256,
        "m": 7, "ell": 6, "eta": 1, "kappa": 60,
        "challenge_bytes": 32, "logq": 31, "nu_b": 11, "nu_w": 26,
        "alpha_coeff": DEFAULT_ALPHA_COEFF,
    },
    {
        "name": "dualms2.0-rounding-n1024", "n": 1024, "N": 256,
        "m": 10, "ell": 6, "eta": 1, "kappa": 60,
        "challenge_bytes": 32, "logq": 32, "nu_b": 11, "nu_w": 28,
        "alpha_coeff": DEFAULT_ALPHA_COEFF,
    },
]

def mlwe_classical_bits(eta, N, ell, q):
    return get_classical_hardness(findMLWEdelta(eta, N, ell, log(q, 2), q))

def msis_classical_bits(gamma_bound, N, m, q):
    return get_classical_hardness(findMSISdelta(gamma_bound, N, m, log(q, 2)))

def profile_result(profile):
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
    ent = RR(1) / 2 + log(sqrt(2 * pi), 2)
    c_bits = challenge_bytes * 8
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

    h_radius = sqrt(m * N) * (sigma * sqrt(2 * n) + eta + kappa * 2 ^ (nu_b - 1) + 2 ^ nu_w) / (2 ^ nu_w)
    h_amp = max(0, floor(h_radius))
    h_bpc = qh_bpc if h_amp >= qh_cap else (0 if h_amp == 0 else ceil(log(2 * h_amp + 1, 2)))
    aggsig_kb = (
        c_bits +
        ell * N * (ent + log(sqrt(n) * sigma, 2)) +
        N * (ent + log(sqrt(n) * sigma, 2)) +
        ell * N * ceil(log(2 * n * eta + 1, 2)) +
        m * N * h_bpc
    ) / KB

    B = sigma * sqrt(2 * n * ell * N) + sqrt(m * N) * (sigma * sqrt(2 * n) + eta + kappa * 2 ^ (nu_b - 1) + 2 ^ nu_w)
    B_n_prime = sigma * sqrt(2 * n * N)
    B_n_double_prime = eta * sqrt(ell * N * n)
    gamma2 = sqrt(B ^ 2 + B_n_prime ^ 2 + B_n_double_prime ^ 2 + (N_STAR ^ 2) * kappa + (2 ^ (2 * nu_b)) * N * m)
    mlwe_c = mlwe_classical_bits(eta, N, ell, q)
    msis_c = msis_classical_bits(gamma2, N, m, q)

    return {
        "profile": profile["name"], "n": n, "q_bits": q_bits,
        "m": m, "ell": ell, "nu_b": nu_b, "nu_w": nu_w,
        "pk_kb": pk_kb, "aggpk_kb": aggpk_kb, "msg1_kb": msg1_kb,
        "sig_i_kb": sig_i_kb, "aggsig_kb": aggsig_kb,
        "pk_plus_sig_kb": aggpk_kb + aggsig_kb, "log_sigma": log(sigma, 2),
        "mlwe_c": mlwe_c, "msis_c": msis_c,
    }

def fmt(x, digits=2):
    return ("%%.%df" % digits) % RR(x).n()

print("DualMS2.0 rounding parameter summary")
print("profile | n | q_bits | m | ell | nu_b | nu_w | pk | aggpk | msg1 | sig | aggsig | aggpk+aggsig | log_sigma | MSIS | MLWE")
for result in [profile_result(profile) for profile in PROFILES]:
    print(
        result["profile"], "|", result["n"], "|", result["q_bits"], "|",
        result["m"], "|", result["ell"], "|", result["nu_b"], "|", result["nu_w"], "|",
        fmt(result["pk_kb"]), "|", fmt(result["aggpk_kb"]), "|", fmt(result["msg1_kb"]), "|",
        fmt(result["sig_i_kb"]), "|", fmt(result["aggsig_kb"]), "|",
        fmt(result["pk_plus_sig_kb"]), "|", fmt(result["log_sigma"]), "|",
        fmt(result["msis_c"], 1), "|", fmt(result["mlwe_c"], 1)
    )
