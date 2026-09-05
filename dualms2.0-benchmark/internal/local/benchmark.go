package local

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type benchmarkRunMetrics struct {
	VerifyOK                   bool
	Attempts                   int
	RejectedAttempts           int
	SetupDuration              time.Duration
	KeyGenDuration             time.Duration
	KeyAggDuration             time.Duration
	Sign1SuccessfulDuration    time.Duration
	Sign1TotalDuration         time.Duration
	Sign2SuccessfulDuration    time.Duration
	Sign2TotalDuration         time.Duration
	SAggDuration               time.Duration
	VerifyDuration             time.Duration
	OnlineDuration             time.Duration
	EndToEndDuration           time.Duration
	KeyGenPerSignerMillis      []float64
	Sign1PerSignerMillis       []float64
	Sign2PerSignerMillis       []float64
	EffectiveRejectionM        float64
	EffectiveTargetRejectionMn float64
}

func BenchmarkRun(cfg Config, runs int) error {
	if runs <= 0 {
		return fmt.Errorf("benchmark runs must be positive")
	}
	if cfg.MaxSigningAttempts <= 0 {
		return fmt.Errorf("max signing attempts must be positive")
	}

	cachedListDigest, err := prepareBenchmarkKeyAggListDigest(cfg)
	if err != nil {
		return fmt.Errorf("prepare cached KeyAgg list digest: %w", err)
	}

	metrics := make([]benchmarkRunMetrics, 0, runs)
	for run := 1; run <= runs; run++ {
		runCfg := cfg
		runCfg.Message = fmt.Sprintf("%s/benchmark-run-%03d", cfg.Message, run)

		fmt.Printf("RUN %d/%d\n", run, runs)
		metric, err := benchmarkOnce(runCfg, run, cachedListDigest)
		if err != nil {
			return fmt.Errorf("benchmark run %d: %w", run, err)
		}
		metrics = append(metrics, metric)
		fmt.Printf("  attempts=%d rejected=%d verify=%v online=%.3f ms end_to_end=%.3f ms\n",
			metric.Attempts,
			metric.RejectedAttempts,
			metric.VerifyOK,
			durationMillis(metric.OnlineDuration),
			durationMillis(metric.EndToEndDuration),
		)
	}

	printBenchmarkSummary(cfg, metrics)
	return nil
}

func prepareBenchmarkKeyAggListDigest(cfg Config) ([]byte, error) {
	pp, err := Setup(cfg)
	if err != nil {
		return nil, err
	}
	keys := make([]PublicKey, 0, cfg.NumSigners)
	for signerID := 0; signerID < cfg.NumSigners; signerID++ {
		pk, _, err := KeyGen(pp, signerID)
		if err != nil {
			return nil, fmt.Errorf("keygen signer %d: %w", signerID, err)
		}
		keys = append(keys, pk)
	}
	return HAggListDigest(pp, keys), nil
}

func benchmarkOnce(cfg Config, run int, cachedListDigest []byte) (benchmarkRunMetrics, error) {
	endToEndStart := time.Now()

	setupStart := time.Now()
	pp, err := Setup(cfg)
	if err != nil {
		return benchmarkRunMetrics{}, err
	}
	setupDuration := time.Since(setupStart)

	signers := make([]*Signer, cfg.NumSigners)
	for i := 0; i < cfg.NumSigners; i++ {
		signers[i] = NewSigner(i)
	}

	keys := make([]PublicKey, 0, len(signers))
	keyGenDurations := make([]float64, 0, len(signers))
	keyGenStart := time.Now()
	for _, signer := range signers {
		start := time.Now()
		pk, sk, err := KeyGen(pp, signer.ID)
		if err != nil {
			return benchmarkRunMetrics{}, err
		}
		keyGenDurations = append(keyGenDurations, durationMillis(time.Since(start)))
		signer.PK = pk
		signer.SK = sk
		keys = append(keys, pk)
	}
	keyGenDuration := time.Since(keyGenStart)

	keyAggStart := time.Now()
	keyAgg, err := KeyAggWithListDigest(pp, keys, cachedListDigest)
	if err != nil {
		return benchmarkRunMetrics{}, err
	}
	keyAggDuration := time.Since(keyAggStart)

	var attempts int
	var rejectedAttempts int
	var totalSign1Duration time.Duration
	var totalSign2Duration time.Duration
	var sign1SuccessfulDuration time.Duration
	var sign2SuccessfulDuration time.Duration
	var sign1PerSignerMillis []float64
	var sign2PerSignerMillis []float64
	var sAggDuration time.Duration
	var verifyDuration time.Duration
	var verifyOK bool

	for attempt := 1; attempt <= cfg.MaxSigningAttempts; attempt++ {
		attempts = attempt
		sessionID := fmt.Sprintf("benchmark-run-%03d-attempt-%03d", run, attempt)
		allMsgs := make([]Round1Msg, 0, len(signers))
		states := make([]Round1State, 0, len(signers))
		attemptSign1PerSigner := make([]float64, 0, len(signers))

		sign1Start := time.Now()
		for _, signer := range signers {
			start := time.Now()
			msg, st, err := Sign1(pp, signer.SK, keyAgg, cfg.Message, sessionID)
			if err != nil {
				return benchmarkRunMetrics{}, err
			}
			attemptSign1PerSigner = append(attemptSign1PerSigner, durationMillis(time.Since(start)))
			allMsgs = append(allMsgs, msg)
			states = append(states, st)
		}
		attemptSign1Duration := time.Since(sign1Start)
		totalSign1Duration += attemptSign1Duration

		partials := make([]PartialSig, 0, len(signers))
		attemptSign2PerSigner := make([]float64, 0, len(signers))
		sign2Start := time.Now()
		for _, st := range states {
			start := time.Now()
			partial, err := Sign2(pp, st, allMsgs)
			if err != nil {
				return benchmarkRunMetrics{}, err
			}
			attemptSign2PerSigner = append(attemptSign2PerSigner, durationMillis(time.Since(start)))
			partials = append(partials, partial)
		}
		attemptSign2Duration := time.Since(sign2Start)
		totalSign2Duration += attemptSign2Duration

		if abortedSignerIDs := collectAbortedSignerIDs(partials); len(abortedSignerIDs) > 0 {
			rejectedAttempts++
			continue
		}

		sign1SuccessfulDuration = attemptSign1Duration
		sign2SuccessfulDuration = attemptSign2Duration
		sign1PerSignerMillis = attemptSign1PerSigner
		sign2PerSignerMillis = attemptSign2PerSigner

		sAggStart := time.Now()
		sig, err := SAgg(pp, cfg.Message, keyAgg.AggPK, keyAgg.SignerIDs, allMsgs, partials)
		if err != nil {
			return benchmarkRunMetrics{}, err
		}
		sAggDuration = time.Since(sAggStart)

		verifyStart := time.Now()
		verifyOK, err = Verify(pp, cfg.Message, sig, keyAgg.AggPK)
		if err != nil {
			return benchmarkRunMetrics{}, err
		}
		verifyDuration = time.Since(verifyStart)
		if !verifyOK {
			return benchmarkRunMetrics{}, fmt.Errorf("verify rejected aggregate signature")
		}
		break
	}

	if !verifyOK {
		return benchmarkRunMetrics{}, fmt.Errorf("signing failed after %d attempts; every attempt had at least one rejected signer", cfg.MaxSigningAttempts)
	}

	onlineDuration := totalSign1Duration + totalSign2Duration + sAggDuration
	return benchmarkRunMetrics{
		VerifyOK:                   verifyOK,
		Attempts:                   attempts,
		RejectedAttempts:           rejectedAttempts,
		SetupDuration:              setupDuration,
		KeyGenDuration:             keyGenDuration,
		KeyAggDuration:             keyAggDuration,
		Sign1SuccessfulDuration:    sign1SuccessfulDuration,
		Sign1TotalDuration:         totalSign1Duration,
		Sign2SuccessfulDuration:    sign2SuccessfulDuration,
		Sign2TotalDuration:         totalSign2Duration,
		SAggDuration:               sAggDuration,
		VerifyDuration:             verifyDuration,
		OnlineDuration:             onlineDuration,
		EndToEndDuration:           time.Since(endToEndStart),
		KeyGenPerSignerMillis:      keyGenDurations,
		Sign1PerSignerMillis:       sign1PerSignerMillis,
		Sign2PerSignerMillis:       sign2PerSignerMillis,
		EffectiveRejectionM:        pp.RejectionM,
		EffectiveTargetRejectionMn: pp.TargetRejectionMn,
	}, nil
}

func printBenchmarkSummary(cfg Config, metrics []benchmarkRunMetrics) {
	fmt.Println()
	fmt.Println("Benchmark summary")
	fmt.Printf("Config: runs=%d signers=%d target M_n=%.6g signer M=%.6g max attempts=%d\n",
		len(metrics),
		cfg.NumSigners,
		metrics[0].EffectiveTargetRejectionMn,
		metrics[0].EffectiveRejectionM,
		cfg.MaxSigningAttempts,
	)
	fmt.Printf("Verification: %d/%d passed\n", countVerified(metrics), len(metrics))
	attempts := sortedCopy(metricAttempts(metrics))
	fmt.Printf("Attempts: mean=%.3f median=%.3f min=%.0f max=%.0f rejected_mean=%.3f\n",
		mean(attempts),
		median(attempts),
		minFloat(attempts),
		maxFloat(attempts),
		mean(metricRejectedAttempts(metrics)),
	)
	fmt.Println()

	keyGenSequentialName := "KeyGen sequential (all signers)"
	if cfg.NumSigners == 32 || cfg.NumSigners == 1024 {
		keyGenSequentialName = fmt.Sprintf("KeyGen sequential (all %d signers)", cfg.NumSigners)
	}
	rows := []benchmarkTableRow{
		newBenchmarkDurationRow("Setup", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.SetupDuration })),
		newBenchmarkFloatRow("KeyGen (per-signer local)", flattenPerSigner(metrics, func(m benchmarkRunMetrics) []float64 { return m.KeyGenPerSignerMillis })),
		newBenchmarkDurationRow(keyGenSequentialName, metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.KeyGenDuration })),
		newBenchmarkDurationRow("KeyAgg", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.KeyAggDuration })),
		newBenchmarkFloatRow("Sign1 (per-signer local)", flattenPerSigner(metrics, func(m benchmarkRunMetrics) []float64 { return m.Sign1PerSignerMillis })),
		newBenchmarkDurationRow("Sign1 successful attempt sequential", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign1SuccessfulDuration })),
		newBenchmarkDurationRow("Sign1 total across attempts", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign1TotalDuration })),
		newBenchmarkFloatRow("Sign2 (per-signer local)", flattenPerSigner(metrics, func(m benchmarkRunMetrics) []float64 { return m.Sign2PerSignerMillis })),
		newBenchmarkDurationRow("Sign2 successful attempt sequential", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign2SuccessfulDuration })),
		newBenchmarkDurationRow("Sign2 total across attempts", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign2TotalDuration })),
		newBenchmarkDurationRow("SAgg", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.SAggDuration })),
		newBenchmarkDurationRow("Verify", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.VerifyDuration })),
		newBenchmarkDurationRow("Online total incl. rejected attempts", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.OnlineDuration })),
		newBenchmarkDurationRow("End-to-end incl. setup/verify", metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.EndToEndDuration })),
	}
	terms := deriveSignTerms(cfg.NumSigners, metrics)
	if cfg.NumSigners == 32 || cfg.NumSigners == 1024 {
		printBenchmarkTableGrouped(rows, terms.TRejectedMean)
	} else {
		printBenchmarkTable(rows)
	}
	printDerivedSignTermSummary(terms)
}

type benchmarkTableRow struct {
	Name   string
	Mean   float64
	Median float64
	Min    float64
	Max    float64
	StdDev float64
}

func newBenchmarkDurationRow(name string, durations []time.Duration) benchmarkTableRow {
	values := make([]float64, 0, len(durations))
	for _, duration := range durations {
		values = append(values, durationMillis(duration))
	}
	return newBenchmarkFloatRow(name, values)
}

func newBenchmarkFloatRow(name string, values []float64) benchmarkTableRow {
	sorted := sortedCopy(values)
	avg := mean(sorted)
	return benchmarkTableRow{
		Name:   name,
		Mean:   avg,
		Median: median(sorted),
		Min:    minFloat(sorted),
		Max:    maxFloat(sorted),
		StdDev: stddev(sorted, avg),
	}
}

func printBenchmarkTable(rows []benchmarkTableRow) {
	fmt.Println("+------------------------------------------+-----------+-----------+-----------+-----------+-----------+")
	fmt.Println("| Phase                                    | Mean ms   | Median ms | Min ms    | Max ms    | StdDev ms |")
	fmt.Println("+------------------------------------------+-----------+-----------+-----------+-----------+-----------+")
	for _, row := range rows {
		fmt.Printf("| %-40s | %9.3f | %9.3f | %9.3f | %9.3f | %9.3f |\n",
			row.Name,
			row.Mean,
			row.Median,
			row.Min,
			row.Max,
			row.StdDev,
		)
	}
	fmt.Println("+------------------------------------------+-----------+-----------+-----------+-----------+-----------+")
}

func printBenchmarkTableGrouped(rows []benchmarkTableRow, rejectedMean float64) {
	const border = "+------------------------------------------+-----------+-----------+-----------+-----------+-----------+"

	fmt.Println(border)
	fmt.Println("| Phase                                    | Mean ms   | Median ms | Min ms    | Max ms    | StdDev ms |")
	fmt.Println(border)
	for index, row := range rows {
		fmt.Printf("| %-40s | %9.3f | %9.3f | %9.3f | %9.3f | %9.3f |\n",
			row.Name,
			row.Mean,
			row.Median,
			row.Min,
			row.Max,
			row.StdDev,
		)
		if index == 3 || index == 9 || index == 11 || index == 13 {
			fmt.Println(border)
		}
	}
	fmt.Printf("| %-40s | %9.3f | %9s | %9s | %9s | %9s |\n",
		"t",
		rejectedMean,
		"-",
		"-",
		"-",
		"-",
	)
	fmt.Println(border)
}

type derivedSignTerms struct {
	TRejectedMean        float64
	Sign1Time1           float64
	Sign1Time2           float64
	Sign1Time1TPlusTime2 float64
	Sign2Time1           float64
	Sign2Time2           float64
	Sign2Time1TPlusTime2 float64
}

func printDerivedSignTermSummary(terms derivedSignTerms) {
	fmt.Println()
	fmt.Println("Derived Sign terms")
	fmt.Printf("  t(rejected_mean)=%.3f\n", terms.TRejectedMean)
	fmt.Printf("  Sign1: time1=%.3f time2=%.3f time1*t+time2=%.3f\n",
		terms.Sign1Time1, terms.Sign1Time2, terms.Sign1Time1TPlusTime2)
	fmt.Printf("  Sign2: time1=%.3f time2=%.3f time1*t+time2=%.3f\n",
		terms.Sign2Time1, terms.Sign2Time2, terms.Sign2Time1TPlusTime2)
	fmt.Printf("DERIVED_SIGN_TERMS t=%.6f sign1_time1=%.6f sign1_time2=%.6f sign1_time1_t_plus_time2=%.6f sign2_time1=%.6f sign2_time2=%.6f sign2_time1_t_plus_time2=%.6f\n",
		terms.TRejectedMean,
		terms.Sign1Time1,
		terms.Sign1Time2,
		terms.Sign1Time1TPlusTime2,
		terms.Sign2Time1,
		terms.Sign2Time2,
		terms.Sign2Time1TPlusTime2,
	)
}

func deriveSignTerms(numSigners int, metrics []benchmarkRunMetrics) derivedSignTerms {
	tRejectedMean := mean(metricRejectedAttempts(metrics))
	sign1Time2 := mean(flattenPerSigner(metrics, func(m benchmarkRunMetrics) []float64 { return m.Sign1PerSignerMillis }))
	sign2Time2 := mean(flattenPerSigner(metrics, func(m benchmarkRunMetrics) []float64 { return m.Sign2PerSignerMillis }))
	sign1SuccessfulSequentialMean := mean(durationMillisSlice(metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign1SuccessfulDuration })))
	sign2SuccessfulSequentialMean := mean(durationMillisSlice(metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign2SuccessfulDuration })))
	sign1TotalMean := mean(durationMillisSlice(metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign1TotalDuration })))
	sign2TotalMean := mean(durationMillisSlice(metricDurations(metrics, func(m benchmarkRunMetrics) time.Duration { return m.Sign2TotalDuration })))

	sign1Time1 := 0.0
	sign2Time1 := 0.0
	if tRejectedMean > 0 && numSigners > 0 {
		sign1Time1 = (sign1TotalMean - sign1SuccessfulSequentialMean) / (tRejectedMean * float64(numSigners))
		sign2Time1 = (sign2TotalMean - sign2SuccessfulSequentialMean) / (tRejectedMean * float64(numSigners))
	}

	return derivedSignTerms{
		TRejectedMean:        tRejectedMean,
		Sign1Time1:           sign1Time1,
		Sign1Time2:           sign1Time2,
		Sign1Time1TPlusTime2: sign1Time1*tRejectedMean + sign1Time2,
		Sign2Time1:           sign2Time1,
		Sign2Time2:           sign2Time2,
		Sign2Time1TPlusTime2: sign2Time1*tRejectedMean + sign2Time2,
	}
}

func metricDurations(metrics []benchmarkRunMetrics, selectDuration func(benchmarkRunMetrics) time.Duration) []time.Duration {
	values := make([]time.Duration, 0, len(metrics))
	for _, metric := range metrics {
		values = append(values, selectDuration(metric))
	}
	return values
}

func flattenPerSigner(metrics []benchmarkRunMetrics, selectValues func(benchmarkRunMetrics) []float64) []float64 {
	values := make([]float64, 0)
	for _, metric := range metrics {
		values = append(values, selectValues(metric)...)
	}
	return values
}

func metricAttempts(metrics []benchmarkRunMetrics) []float64 {
	values := make([]float64, 0, len(metrics))
	for _, metric := range metrics {
		values = append(values, float64(metric.Attempts))
	}
	return values
}

func metricRejectedAttempts(metrics []benchmarkRunMetrics) []float64 {
	values := make([]float64, 0, len(metrics))
	for _, metric := range metrics {
		values = append(values, float64(metric.RejectedAttempts))
	}
	return values
}

func countVerified(metrics []benchmarkRunMetrics) int {
	count := 0
	for _, metric := range metrics {
		if metric.VerifyOK {
			count++
		}
	}
	return count
}

func sortedCopy(values []float64) []float64 {
	copied := append([]float64(nil), values...)
	sort.Float64s(copied)
	return copied
}

func minFloat(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	return values[0]
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	return values[len(values)-1]
}

func mean(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func median(values []float64) float64 {
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func stddev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, value := range values {
		diff := value - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(values)))
}

func durationMillisSlice(durations []time.Duration) []float64 {
	values := make([]float64, 0, len(durations))
	for _, duration := range durations {
		values = append(values, durationMillis(duration))
	}
	return values
}
