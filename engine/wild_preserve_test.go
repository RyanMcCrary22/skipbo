package engine

import "testing"

func TestBuildingPile_PreservesWild(t *testing.T) {
	bp := &BuildingPile{}
	
	err := bp.Play(NewCard(1))
	if err != nil {
		t.Fatal(err)
	}
	
	// Play a SkipBo acting as a 2
	sb := NewCard(SkipBo)
	err = bp.Play(sb)
	if err != nil {
		t.Fatal(err)
	}
	
	if bp.TopValue() != 2 {
		t.Fatalf("expected TopValue to be 2, got %d", bp.TopValue())
	}
	
	// Fill the rest to clear it (3 through 12)
	for i := CardValue(3); i <= 12; i++ {
		bp.Play(NewCard(i))
	}
	
	if !bp.IsComplete() {
		t.Fatal("expected pile to be complete")
	}
	
	cleared := bp.Clear()
	if len(cleared) != 12 {
		t.Fatalf("expected 12 cleared cards, got %d", len(cleared))
	}
	
	// The second card should still be a SkipBo, NOT a 2!
	// This proves that wilds are safely recycled.
	if cleared[1].Value != SkipBo {
		t.Errorf("expected cleared card to remain SkipBo, got %v", cleared[1].Value)
	}
}
