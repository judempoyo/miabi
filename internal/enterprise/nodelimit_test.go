// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package enterprise

import "testing"

func TestNodeLimitResolution(t *testing.T) {
	cases := []struct {
		name string
		ent  Entitlements
		want int
	}{
		{"community default (no license)", Entitlements{Edition: EditionCommunity}, CommunityNodeLimit},
		{"empty edition treated as community", Entitlements{}, CommunityNodeLimit},
		{"paid, no explicit cap is unlimited", Entitlements{Edition: EditionEnterprise}, -1},
		{"explicit license cap wins over edition default",
			Entitlements{Edition: EditionEnterprise, Limits: map[string]int{LimitNodeLimit: 10}}, 10},
		{"explicit cap also overrides community default",
			Entitlements{Edition: EditionCommunity, Limits: map[string]int{LimitNodeLimit: 1}}, 1},
		{"explicit unlimited on a license",
			Entitlements{Edition: EditionEnterprise, Limits: map[string]int{LimitNodeLimit: -1}}, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ent.NodeLimit(); got != tc.want {
				t.Errorf("NodeLimit() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPlanLimitResolution(t *testing.T) {
	cases := []struct {
		name string
		ent  Entitlements
		want int
	}{
		{"community default caps at the seeded plans", Entitlements{Edition: EditionCommunity}, CommunityPlanLimit},
		{"empty edition treated as community", Entitlements{}, CommunityPlanLimit},
		{"paid, no explicit cap is unlimited", Entitlements{Edition: EditionEnterprise}, -1},
		{"explicit license cap wins over edition default",
			Entitlements{Edition: EditionEnterprise, Limits: map[string]int{LimitPlanLimit: 10}}, 10},
		{"explicit cap also overrides community default (tiered license)",
			Entitlements{Edition: EditionEnterprise, Limits: map[string]int{LimitPlanLimit: 5}}, 5},
		{"explicit unlimited on a license",
			Entitlements{Edition: EditionEnterprise, Limits: map[string]int{LimitPlanLimit: -1}}, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ent.PlanLimit(); got != tc.want {
				t.Errorf("PlanLimit() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTierPresets(t *testing.T) {
	if len(Tiers) == 0 {
		t.Fatal("no tiers defined")
	}
	for _, tier := range Tiers {
		// Every preset flag must be a known entitlement (guards preset typos).
		for _, f := range tier.Flags {
			if !IsKnownFlag(f) {
				t.Errorf("tier %q references unknown flag %q", tier.Name, f)
			}
		}
		// Presets set both bounded limits.
		if _, ok := tier.Limits[LimitNodeLimit]; !ok {
			t.Errorf("tier %q missing node_limit", tier.Name)
		}
		if _, ok := tier.Limits[LimitPlanLimit]; !ok {
			t.Errorf("tier %q missing plan_limit", tier.Name)
		}
		// TierByName resolves (case-insensitively).
		if got, ok := TierByName(tier.Name); !ok || got.Name != tier.Name {
			t.Errorf("TierByName(%q) failed", tier.Name)
		}
	}
	// Enterprise unlocks every flag and is unlimited.
	ent, _ := TierByName(TierEnterprise)
	if len(ent.Flags) != len(AllFlags) {
		t.Errorf("enterprise tier has %d flags, want all %d", len(ent.Flags), len(AllFlags))
	}
	if ent.Limits[LimitNodeLimit] != -1 || ent.Limits[LimitPlanLimit] != -1 {
		t.Error("enterprise tier should be unlimited")
	}
	if _, ok := TierByName("nope"); ok {
		t.Error("TierByName should not resolve an unknown tier")
	}
}
