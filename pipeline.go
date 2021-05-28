// Code from https://pkg.go.dev/github.com/google/differential-privacy/privacy-on-beam/pbeam?utm_source=godoc (Modified)
package main

import (
	"context"
	"fmt"

	"github.com/apache/beam/sdks/go/pkg/beam"
	"github.com/apache/beam/sdks/go/pkg/beam/io/textio"
	"github.com/apache/beam/sdks/go/pkg/beam/runners/direct"
	"github.com/google/differential-privacy/privacy-on-beam/pbeam"
)


func main() {
	// visit contains the data corresponding to a single restaurant visit.
	type visit struct {
		visitorID  string
		eurosSpent int
		weekday    int
	}

	// Initialize the pipeline.
	beam.Init()
	p := beam.NewPipeline()
	s := p.Root()


	// Load the data and parse each visit, ignoring parsing errors.
	icol := textio.Read(s, "week_data.csv")
	icol = beam.ParDo(s, func(s string, emit func(visit)) {
		var visitorID string
		var euros, weekday int
		_, err := fmt.Sscanf(s, "%s, %d, %d", &visitorID, &euros, &weekday)
		if err != nil {
			return
		}
		emit(visit{visitorID, euros, weekday})
	}, icol)

	// Transform the input PCollection into a PrivatePCollection.

	// ε and δ are the differential privacy parameters that quantify the privacy
	// provided by the pipeline.
	const ε, δ = 1, 1e-3

	privacySpec := pbeam.NewPrivacySpec(ε, δ)
	pcol := pbeam.MakePrivateFromStruct(s, icol, privacySpec, "visitorID")
	// pcol is now a PrivatePCollection&lt;visit&gt;.

	// Compute the differentially private sum-up revenue per weekday. To do so,
	// we extract a KV pair, where the key is weekday and the value is the
	// money spent.
	pWeekdayEuros := pbeam.ParDo(s, func(v visit) (int, int) {
		return v.weekday, v.eurosSpent
	}, pcol)
	sumParams := pbeam.SumParams{
		// There is only a single differentially private aggregation in this
		// pipeline, so the entire privacy budget will be consumed (ε=1 and
		// δ=10⁻³). If multiple aggregations are present, we would need to
		// manually specify the privacy budget used by each.

		// If a visitor of the restaurant is present in more than 4 weekdays,
		// some of these contributions will be randomly dropped.
		// Larger values lets you keep more contributions (more of the raw data)
		// but lead to more noise in the output because the noise will be scaled
		// by the value. See the relevant section in the codelab for details:
		// https://codelabs.developers.google.com/codelabs/privacy-on-beam/#8
		MaxPartitionsContributed: 4,

		// If a visitor of the restaurant spends more than 50 euros, or less
		// than 0 euros, their contribution will be clamped.
		// Similar to MaxPartitionsContributed, a larger interval lets you keep more
		// of the raw data but lead to more noise in the output because the noise
		// will be scaled by max(|MinValue|,|MaxValue|).
		MinValue: 0,
		MaxValue: 50,
	}
	ocol := pbeam.SumPerKey(s, pWeekdayEuros, sumParams)

	// ocol is a regular PCollection; it can be written to disk.
	formatted := beam.ParDo(s, func(weekday int, sum int64) string {
		return fmt.Sprintf("Weekday n°%d: total spend is %d euros", weekday, sum)
	}, ocol)
	textio.Write(s, "spend_per_weekday.txt", formatted)

	// Execute the pipeline.
	if _, err := direct.Execute(context.Background(), p); err != nil {
		fmt.Printf("Pipeline failed: %v", err)
	}
}
