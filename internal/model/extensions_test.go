package model

import "testing"

func TestExtensionClassification(t *testing.T) {
	anim := []string{"zani", ".zani", "ZAN", ".zan", "zcut", ".ZCUT"}
	for _, e := range anim {
		if !IsAnimationExt(e) {
			t.Errorf("%q should be an animation ext", e)
		}
		if IsBundleExt(e) {
			t.Errorf("%q should not be a bundle ext", e)
		}
	}
	bundle := []string{"zbun", ".zbun", "ZBU", ".zbu"}
	for _, e := range bundle {
		if !IsBundleExt(e) {
			t.Errorf("%q should be a bundle ext", e)
		}
		if IsAnimationExt(e) {
			t.Errorf("%q should not be an animation ext", e)
		}
	}
	if IsAnimationExt("png") || IsBundleExt("scr") {
		t.Error("unrelated extensions misclassified")
	}
}

func TestSaveFormExtensions(t *testing.T) {
	if AnimationExt(SaveFormLong) != "zani" || AnimationExt(SaveForm83) != "zan" {
		t.Error("animation save-form extensions wrong")
	}
	if BundleExt(SaveFormLong) != "zbun" || BundleExt(SaveForm83) != "zbu" {
		t.Error("bundle save-form extensions wrong")
	}
}

func TestAnimationAliasesLoadAsZCUT(t *testing.T) {
	s := New(16, 16)
	for f := 0; f < s.FrameCount(); f++ {
		s.Select(f)
		s.Set(f%16, 0, true)
	}
	s.Select(0)
	data, _ := s.MarshalZCUT()
	for _, ext := range []string{"zani", "zan", "zcut"} {
		sp, err := LoadByExtension(ext, data)
		if err != nil {
			t.Fatalf("%s: %v", ext, err)
		}
		if sp.FrameCount() != s.FrameCount() {
			t.Errorf("%s: loaded %d frames, want %d", ext, sp.FrameCount(), s.FrameCount())
		}
	}
}
