package evidence

import (
	"context"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestEvidenceOnlyPromotion(t *testing.T) {
	finding := completeFinding()
	promoted, result, err := DecidePromotion(context.Background(), BaselineValidator{}, PromotionProposal{Finding: finding, EvidenceOnly: true, Signals: []Signal{SignalReproduction, SignalImpact, SignalControl}})
	if err != nil || !result.Allowed || promoted.State != domain.FindingVerified {
		t.Fatalf("finding=%#v result=%#v err=%v", promoted, result, err)
	}
}
func TestWeakSignalsNeverProveFindingAlone(t *testing.T) {
	for _, signal := range []Signal{SignalHTTPStatus, SignalScannerMatch, SignalCodePattern, SignalModelAssertion} {
		t.Run(string(signal), func(t *testing.T) {
			finding := completeFinding()
			promoted, result, err := DecidePromotion(context.Background(), BaselineValidator{}, PromotionProposal{Finding: finding, EvidenceOnly: true, Signals: []Signal{signal}})
			if err != nil || result.Allowed || promoted.State == domain.FindingVerified {
				t.Fatalf("signal=%s finding=%#v result=%#v err=%v", signal, promoted, result, err)
			}
		})
	}
}
func TestNonEvidenceTurnDowngraded(t *testing.T) {
	finding := completeFinding()
	_, result, _ := DecidePromotion(context.Background(), BaselineValidator{}, PromotionProposal{Finding: finding, Signals: []Signal{SignalReproduction, SignalImpact, SignalControl}})
	if result.Allowed {
		t.Fatal("normal model turn promoted finding")
	}
}
