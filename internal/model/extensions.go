package model

// File extensions zenimate uses. Two conceptual types, each with a long form
// and an 8.3 form for old filesystems:
//
//	Animation (one animated sprite, ZCUT-encoded): .zani / .zan
//	Bundle    (a zip of many .zani animations):    .zbun / .zbu
//
// The animation bytes are exactly the ZCUT format, so .zani is a rename of the
// former .zcut, not a new encoding; raw .zcut is still accepted on load as an
// alias for interoperability with the wider toolchain.
const (
	ExtAnimation    = "zani" // long form, single animated sprite
	ExtAnimation83  = "zan"  // 8.3 form of the above
	ExtAnimationOld = "zcut" // legacy/interop alias, still accepted on load
	ExtBundle       = "zbun" // long form, a collection of .zani animations
	ExtBundle83     = "zbu"  // 8.3 form of the above
)

// SaveForm selects which extension spelling zenimate writes when saving.
type SaveForm int

const (
	// SaveFormLong writes the long extensions (.zani, .zbun).
	SaveFormLong SaveForm = iota
	// SaveForm83 writes the 8.3 extensions (.zan, .zbu).
	SaveForm83
)

// AnimationExt returns the animation extension to save with, for the given form.
func AnimationExt(form SaveForm) string {
	if form == SaveForm83 {
		return ExtAnimation83
	}
	return ExtAnimation
}

// BundleExt returns the bundle extension to save with, for the given form.
func BundleExt(form SaveForm) string {
	if form == SaveForm83 {
		return ExtBundle83
	}
	return ExtBundle
}

// IsAnimationExt reports whether ext (any case, optional leading dot) names a
// single animated sprite zenimate can open: .zani, .zan, or the .zcut alias.
func IsAnimationExt(ext string) bool {
	switch normaliseExt(ext) {
	case ExtAnimation, ExtAnimation83, ExtAnimationOld:
		return true
	}
	return false
}

// IsBundleExt reports whether ext names a bundle: .zbun or .zbu.
func IsBundleExt(ext string) bool {
	switch normaliseExt(ext) {
	case ExtBundle, ExtBundle83:
		return true
	}
	return false
}

// AnimationExtensions lists every extension accepted for a single animation
// (long, 8.3, and the legacy alias), for dialog filters.
func AnimationExtensions() []string {
	return []string{ExtAnimation, ExtAnimation83, ExtAnimationOld}
}

// BundleExtensions lists every extension accepted for a bundle.
func BundleExtensions() []string {
	return []string{ExtBundle, ExtBundle83}
}
