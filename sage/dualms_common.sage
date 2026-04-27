def findMSISdelta(B, n, d, logq):
    logC = log(B, 2)
    logdelta = logC ^ 2 / (4 * n * d * logq)
    return 2 ^ logdelta


def findMLWEdelta(nu, n, d, logq, q):
    load("lwe_estimator.py")
    n = n * d
    stddev = sqrt(((2 * nu + 1) ^ 2 - 1) / 12)
    alpha = alphaf(sigmaf(stddev), q)
    set_verbose(0)
    L = estimate_lwe(n, alpha, q, reduction_cost_model=BKZ.enum)
    delta_enum1 = L['usvp']['delta_0'] 
    delta_enum2 = L['dec']['delta_0']  
    delta_enum3 = L['dual']['delta_0']  
    L = estimate_lwe(n, alpha, q, reduction_cost_model=BKZ.sieve)
    delta_sieve1 = L['usvp']['delta_0'] 
    delta_sieve2 = L['dec']['delta_0']  
    delta_sieve3 = L['dual']['delta_0']
    return max(delta_enum1, delta_enum2, delta_enum3, delta_sieve1, delta_sieve2, delta_sieve3)
 

def get_classical_hardness(delta):
    from sage.symbolic.constants import pi, e
    right_side_value = delta
    beta = var('beta')
    equation = (((beta / (2 * pi * exp(1))) * ((pi * beta) ^ (1 / beta))) ^ (1 / (2 * (beta - 1))) == right_side_value)
    solutions = find_root(equation, 1, 1000000)
    return solutions * 0.292


def q_bit_length(q):
    return ceil(log(q, 2))


def find_prime_q_below_power(log_q_bits, modulus=8, residue=5, rounding_nus=None):
    p = previous_prime(Integer(floor(RR(2) ** RR(log_q_bits))))
    while True:
        residue_ok = p % modulus == residue
        if rounding_nus is None:
            rounding_ok = True
        else:
            rounding_ok = all(
                p % (2 ^ nu) <= 2 ^ (nu - 1) - 1 for nu in rounding_nus
            )
        if residue_ok and rounding_ok:
            return p
        p = previous_prime(p)
