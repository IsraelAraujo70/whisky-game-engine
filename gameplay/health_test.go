package gameplay

import "testing"

func TestHealthDamageStartsInvulnerability(t *testing.T) {
	h := NewHealth(3)
	h.InvulnerableFor = 0.5

	if !h.Damage(1) {
		t.Fatal("expected damage to apply")
	}
	if h.Current != 2 {
		t.Fatalf("expected hp=2, got %d", h.Current)
	}
	if !h.Invulnerable() {
		t.Fatal("expected invulnerability after damage")
	}
	if h.Damage(1) {
		t.Fatal("expected invulnerability to block repeated damage")
	}

	if err := h.Update(nil, 0.5); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if h.Invulnerable() {
		t.Fatal("expected invulnerability to expire after Update")
	}
	if !h.Damage(1) {
		t.Fatal("expected damage after invulnerability expires")
	}
}

func TestHealthHealClampsToMax(t *testing.T) {
	h := NewHealth(4)
	h.Damage(2)

	if !h.Heal(10) {
		t.Fatal("expected heal to apply")
	}
	if h.Current != 4 {
		t.Fatalf("expected hp=4, got %d", h.Current)
	}
}

func TestHealthFraction(t *testing.T) {
	h := NewHealth(5)
	h.Damage(2)

	if got := h.Fraction(); got != 0.6 {
		t.Fatalf("expected fraction=0.6, got %.2f", got)
	}
}
